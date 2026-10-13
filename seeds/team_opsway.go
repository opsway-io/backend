package seeds

import (
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/utils/pointer"
)

func TeamOpsway(db *gorm.DB) {
	// Team
	t := entities.Team{
		Name: "opsway",
	}
	if err := db.Where(entities.Team{Name: t.Name}).FirstOrCreate(&t).Error; err != nil {
		panic(err)
	}

	// Users
	user := entities.User{
		Email:       "admin@opsway.io",
		Name:        "Douglas Adams",
		DisplayName: pointer.String("Ford Prefect"),
	}

	user.SetPassword("pass")

	if err := db.Where(entities.User{Email: user.Email}).FirstOrCreate(&user).Error; err != nil {
		panic(err)
	}

	// Membership, upserted so re-running the seed fixes the role
	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "team_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "updated_at"}),
	}).Create(&entities.TeamUser{
		UserID: user.ID,
		TeamID: t.ID,
		Role:   entities.TeamRoleAdmin,
	}).Error
	if err != nil {
		panic(err)
	}
}
