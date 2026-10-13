package seeds

import (
	"context"
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Monitors(db *gorm.DB) {
	// Teams
	t := entities.Team{
		Name: "opsway",
	}
	db.FirstOrCreate(&t)

	repo := monitor.NewRepository(db)
	ctx := context.Background()

	// Monitors (created inactive: seeding does not touch the probe schedule)
	for i := 0; i < 30; i++ {
		var m *entities.Monitor
		if i%5 == 0 {
			m = seedMultiStepMonitor(t.ID, i)
		} else {
			m = seedSingleStepMonitor(t.ID)
		}

		if err := repo.Create(ctx, m); err != nil {
			logrus.WithError(err).Error("failed to seed monitor")
		}
	}
}

func seedSettings() entities.MonitorSettings {
	s := entities.MonitorSettings{
		Frequency: time.Minute,
		Auth:      entities.MonitorSettingsAuth{Method: entities.MonitorAuthMethodNone},
		Locations: []string{monitor.DefaultLocation},
	}

	return s
}

func seedSingleStepMonitor(teamID uint) *entities.Monitor {
	return &entities.Monitor{
		TeamID:   teamID,
		Name:     gofakeit.Word(),
		Settings: seedSettings(),
		Steps: []entities.MonitorStep{
			{
				Name:   "Step 1",
				Method: "GET",
				URL:    gofakeit.URL(),
				Body:   entities.MonitorSettingsBody{Type: "NONE"},
				Assertions: []entities.MonitorAssertion{
					{Source: "STATUS_CODE", Operator: "EQUAL", Target: "200"},
				},
			},
		},
	}
}

func seedMultiStepMonitor(teamID uint, i int) *entities.Monitor {
	body := []byte(fmt.Sprintf(`{"name": "seed-%d"}`, i))

	return &entities.Monitor{
		TeamID:   teamID,
		Name:     fmt.Sprintf("Create and fetch %s", gofakeit.Word()),
		Settings: seedSettings(),
		Steps: []entities.MonitorStep{
			{
				Name:    "Create resource",
				Method:  "POST",
				URL:     "https://httpbin.org/anything",
				Headers: []entities.MonitorSettingsHeader{{Key: "Accept", Value: "application/json"}},
				Body:    entities.MonitorSettingsBody{Type: "JSON", Content: &body},
				Assertions: []entities.MonitorAssertion{
					{Source: "STATUS_CODE", Operator: "EQUAL", Target: "200"},
					{Source: "JSON_BODY", Property: "$.json.name", Operator: "NOT_EMPTY"},
				},
				Variables: []entities.MonitorStepVariable{
					{Name: "NAME", Source: "JSON_BODY", Property: "$.json.name"},
				},
			},
			{
				Name:   "Fetch resource",
				Method: "GET",
				URL:    "https://httpbin.org/anything/{{NAME}}",
				Body:   entities.MonitorSettingsBody{Type: "NONE"},
				Assertions: []entities.MonitorAssertion{
					{Source: "STATUS_CODE", Operator: "EQUAL", Target: "200"},
					{Source: "RESPONSE_TIME", Property: "TOTAL", Operator: "LESS_THAN", Target: "5000"},
				},
			},
		},
	}
}
