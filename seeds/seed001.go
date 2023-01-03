package seeds

import (
	"fmt"
	"time"

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

	// Monitors
	m := &entities.Monitor{
		Name: "opsway.io",
		Settings: entities.MonitorSettings{
			Method:    "GET",
			URL:       "https://opsway.io",
			Frequency: time.Minute,
		},
		TeamID: t.ID,
	}
	db.FirstOrCreate(m)

	// Users
	users := []entities.User{
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

	for i := range users {
		u := users[i]

		u.SetPassword("pass")
		fmt.Println(db.Create(&u).Error)

		db.Create(&entities.TeamUser{
			UserID: u.ID,
			TeamID: t.ID,
			Role:   entities.TeamRoleOwner,
		})
	}
}
