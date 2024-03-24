package seeds

import (
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"k8s.io/utils/pointer"
)

func TeamOpsway(db *gorm.DB) {
	// Team
	t := entities.Team{
		Name: "opsway",
	}
	db.FirstOrCreate(&t)

	// Users
	user := entities.User{
		Email:       "admin@opsway.io",
		Name:        "Douglas Adams",
		DisplayName: pointer.StringPtr("Ford Prefect"),
		Teams: []entities.Team{
			t,
		},
	}

	user.SetPassword("pass")

	result := db.Create(&user)
	if result.Error != nil {
		panic(result.Error)
	}
}
