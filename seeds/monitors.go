package seeds

import (
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

func Monitors(db *gorm.DB) {
	// Teams
	t := entities.Team{
		Name: "opsway",
	}
	db.FirstOrCreate(&t)

	// Monitors
	for i := 0; i < 30; i++ {
		m := &entities.Monitor{
			Name: gofakeit.Word(),
			Settings: entities.MonitorSettings{
				Method:    "GET",
				URL:       gofakeit.URL(),
				Frequency: time.Minute,
			},
			TeamID: t.ID,
		}
		db.Create(m)
	}
}
