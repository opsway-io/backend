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
		Email:       "admin@opsway.eu",
		Name:        "Douglas Adams",
		DisplayName: pointer.String("Ford Prefect"),
		Teams: []entities.Team{
			t,
		},
	}

	_ = user.SetPassword("pass")

	result := db.Where(entities.User{Email: user.Email}).FirstOrCreate(&user)
	if result.Error != nil {
		panic(result.Error)
	}

	db.Where(entities.TeamUser{UserID: user.ID, TeamID: t.ID}).FirstOrCreate(&entities.TeamUser{
		UserID: user.ID,
		TeamID: t.ID,
		Role:   entities.TeamRoleAdmin,
	})
}
