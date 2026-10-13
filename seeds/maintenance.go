package seeds

import (
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"k8s.io/utils/pointer"
)

// Maintenance seeds an active, a scheduled and an ended maintenance window for the
// opsway team (run the team_opsway and monitors seeds first). Idempotent per title.
func Maintenance(db *gorm.DB) {
	t := entities.Team{Name: "opsway"}
	if err := db.Where(entities.Team{Name: t.Name}).FirstOrCreate(&t).Error; err != nil {
		panic(err)
	}

	var monitors []entities.Monitor
	if err := db.Where("team_id = ?", t.ID).Order("id ASC").Limit(3).Find(&monitors).Error; err != nil {
		panic(err)
	}

	pick := func(from, to int) []entities.Monitor {
		res := []entities.Monitor{}
		for i := from; i < to && i < len(monitors); i++ {
			res = append(res, monitors[i])
		}

		return res
	}

	now := time.Now().UTC().Truncate(time.Minute)

	windows := []entities.Maintenance{
		{
			Title:       "Database upgrade",
			Description: pointer.String("Upgrading the primary database cluster. Short periods of read-only mode are expected."),
			Settings:    entities.MaintenanceSettings{StartAt: now.Add(-30 * time.Minute), EndAt: now.Add(90 * time.Minute)},
			Monitors:    pick(0, 2),
		},
		{
			Title:       "Network maintenance",
			Description: pointer.String("Our network provider is replacing core switches."),
			Settings:    entities.MaintenanceSettings{StartAt: now.Add(72 * time.Hour), EndAt: now.Add(75 * time.Hour)},
			Monitors:    []entities.Monitor{},
		},
		{
			Title:       "Certificate rotation",
			Description: nil,
			Settings:    entities.MaintenanceSettings{StartAt: now.Add(-5 * 24 * time.Hour), EndAt: now.Add(-5*24*time.Hour + time.Hour)},
			Monitors:    pick(2, 3),
		},
	}

	for i := range windows {
		w := windows[i]
		w.TeamID = t.ID

		var count int64
		if err := db.Model(&entities.Maintenance{}).Where("team_id = ? AND title = ?", t.ID, w.Title).Count(&count).Error; err != nil {
			panic(err)
		}
		if count > 0 {
			continue
		}

		monitors := w.Monitors
		w.Monitors = nil

		if err := db.Omit("Monitors").Create(&w).Error; err != nil {
			panic(err)
		}

		if len(monitors) > 0 {
			if err := db.Model(&w).Association("Monitors").Replace(monitors); err != nil {
				panic(err)
			}
		}
	}
}
