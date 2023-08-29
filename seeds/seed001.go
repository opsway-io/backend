package seeds

import (
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"k8s.io/utils/pointer"
)

func Seed001(db *gorm.DB) {
	// Teams
	t := entities.Team{
		Name: "opsway",
	}
	db.FirstOrCreate(&t)

	// Billing
	b := entities.Billing{
		PaymentPlan: entities.PaymentPlanFree,
		TeamID:      t.ID,
	}
	db.Create(b)

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

	// Users
	defaultUsers := []entities.User{
		{
			Name:        "Douglas Adams",
			DisplayName: pointer.String("I Am Admin"),
			Email:       "admin@opsway.io",
		},
		{
			Name:        "John Doe",
			DisplayName: pointer.String("John"),
			Email:       "john@opsway.io",
		},
		{
			Name:        "Jane Doe",
			DisplayName: pointer.String("Jane"),
			Email:       "jane@opsway.io",
		},
	}

	for i := range defaultUsers {
		u := defaultUsers[i]

		u.SetPassword("pass")
		db.Create(&u)

		db.Create(&entities.TeamUser{
			UserID: u.ID,
			TeamID: t.ID,
			Role:   entities.TeamRoleOwner,
		})
	}

	// Random users

	for i := 0; i < 30; i++ {
		u := entities.User{
			Name:        gofakeit.Name(),
			DisplayName: pointer.String(gofakeit.Username()),
			Email:       gofakeit.Email(),
		}

		u.SetPassword("pass")
		db.Create(&u)

		db.Create(&entities.TeamUser{
			UserID: u.ID,
			TeamID: t.ID,
			Role:   entities.TeamRoleMember,
		})
	}
}
