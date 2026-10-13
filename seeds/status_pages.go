package seeds

import (
	"github.com/lib/pq"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

// StatusPages seeds two status pages for the opsway team using its existing monitors
// (run the team_opsway and monitors seeds first). Idempotent per domain.
func StatusPages(db *gorm.DB) {
	t := entities.Team{Name: "opsway"}
	if err := db.Where(entities.Team{Name: t.Name}).FirstOrCreate(&t).Error; err != nil {
		panic(err)
	}

	var monitorIDs []int64
	if err := db.Model(&entities.Monitor{}).
		Where("team_id = ?", t.ID).
		Order("id ASC").
		Limit(8).
		Pluck("id", &monitorIDs).Error; err != nil {
		panic(err)
	}

	take := func(from, to int) pq.Int64Array {
		res := pq.Int64Array{}
		for i := from; i < to && i < len(monitorIDs); i++ {
			res = append(res, monitorIDs[i])
		}

		return res
	}

	pages := []entities.StatusPage{
		{
			TeamID:       t.ID,
			Name:         "Opsway public status",
			Domain:       "status.opsway.eu",
			Layout:       string(entities.StatusPageLayoutStats),
			ShowBranding: true,
			SupportURL:   "mailto:support@opsway.io",
			MonitorIDs:   take(0, 2),
			Groups: []entities.StatusPageGroup{
				{Name: "API", Order: 0, MonitorIDs: take(2, 5)},
				{Name: "Website", Order: 1, MonitorIDs: take(5, 8)},
			},
		},
		{
			TeamID:       t.ID,
			Name:         "Internal services",
			Domain:       "internal-status.opsway.eu",
			Layout:       string(entities.StatusPageLayoutCompact),
			ShowBranding: false,
			MonitorIDs:   take(0, 4),
			Groups:       []entities.StatusPageGroup{},
		},
	}

	for i := range pages {
		p := pages[i]

		var existing entities.StatusPage
		err := db.Where("domain = ?", p.Domain).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			panic(err)
		}

		if err := db.Create(&p).Error; err != nil {
			panic(err)
		}

		// show_branding has a DB default of true, so a false value is skipped on insert.
		if err := db.Model(&p).Update("show_branding", p.ShowBranding).Error; err != nil {
			panic(err)
		}
	}
}
