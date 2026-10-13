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
			Name:   gofakeit.Word(),
			TeamID: t.ID,
			Settings: entities.MonitorSettings{
				Frequency: time.Minute * 1,
			},
			Steps: []entities.MonitorStep{
				{
					OrderIndex: 0,
					Name:       "Ping",
					Method:     "GET",
					URL:        "https://opsway.io",
					Assertions: []entities.MonitorAssertion{
						{
							Source:   "STATUS_CODE",
							Operator: "EQUAL",
							Target:   "200",
						},
					},
				},
			},
		}
		db.Create(m)
	}
}
