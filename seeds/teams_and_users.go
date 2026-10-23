package seeds

import (
	"github.com/brianvoe/gofakeit"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"k8s.io/utils/pointer"
)

func TeamsAndUsers(db *gorm.DB) {
	// Teams
	t := entities.Team{
		Name: "opsway",
	}
	db.FirstOrCreate(&t)

	// // Monitors
	// for i := 0; i < 30; i++ {
	// 	m := &entities.Monitor{
	// 		Name: gofakeit.Word(),
	// 		Settings: entities.MonitorSettings{
	// 			Method:    "GET",
	// 			URL:       gofakeit.URL(),
	// 			Frequency: time.Minute,
	// 		},
	// 		TeamID: t.ID,
	// 	}
	// 	db.Create(m)
	// }

	// Users
	defaultUsers := []entities.User{
		{
			Name:        "Douglas Adams",
			DisplayName: pointer.String("I Am Admin"),
			Email:       "admin@opsway.eu",
		},
		{
			Name:        "John Doe",
			DisplayName: pointer.String("John"),
			Email:       "john@opsway.eu",
		},
		{
			Name:        "Jane Doe",
			DisplayName: pointer.String("Jane"),
			Email:       "jane@opsway.eu",
		},
	}

	for i := range defaultUsers {
		u := defaultUsers[i]

		_ = u.SetPassword("pass")
		db.Where(entities.User{Email: u.Email}).FirstOrCreate(&u)

		db.Where(entities.TeamUser{UserID: u.ID, TeamID: t.ID}).FirstOrCreate(&entities.TeamUser{
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

		_ = u.SetPassword("pass")
		db.Create(&u)

		db.Create(&entities.TeamUser{
			UserID: u.ID,
			TeamID: t.ID,
			Role:   entities.TeamRoleMember,
		})
	}

	var ruleCount int64
	db.Model(&entities.AlertRule{}).Where("team_id = ?", t.ID).Count(&ruleCount)
	if ruleCount == 0 {
		_ = db.Create(&entities.AlertRule{
			TeamID:    t.ID,
			Name:      "All Incidents",
			Condition: "*",
			Channels:  `["email"]`,
			Enabled:   true,
		}).Error
	}

	// Default Umbrella Platform Entitlements for default team
	for _, prod := range []entities.ProductSlug{
		entities.ProductMonitoring,
		entities.ProductLogSearch,
		entities.ProductPantomath,
		entities.ProductManagedAgent,
	} {
		var ent entities.TeamEntitlement
		db.Where(entities.TeamEntitlement{TeamID: t.ID, Product: prod}).FirstOrCreate(&ent, entities.TeamEntitlement{
			TeamID:       t.ID,
			Product:      prod,
			Tier:         entities.EntitlementTierFree,
			Enabled:      true,
			MonthlyQuota: entities.DefaultMonthlyQuota(prod, entities.EntitlementTierFree),
		})
	}

	// Default Agent Config for default team
	var agentCfg entities.AgentConfig
	db.Where(entities.AgentConfig{TeamID: t.ID}).FirstOrCreate(&agentCfg, entities.AgentConfig{
		TeamID:                 t.ID,
		Enabled:                true,
		ExecutionMode:          entities.AgentModeManaged,
		AutoTriggerOnConsensus: true,
		AutoTriggerOnAnomaly:   true,
		AutoTriggerOnLogSpike:  true,
		MinOccurrences:         1,
		CooldownMinutes:        30,
		RemediationLevel:       entities.RemediationApprovalRequired,
	})
}
