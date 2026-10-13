package seeds

import (
	"time"

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

	// Monitors
	monitorsToSeed := []entities.Monitor{
		{
			Name:   "API",
			TeamID: t.ID,
			State:  entities.MonitorStateActive,
			Settings: entities.MonitorSettings{
				Frequency: time.Minute * 1,
			},
			Steps: []entities.MonitorStep{
				{
					OrderIndex: 0,
					Name:       "Ping",
					Method:     "GET",
					URL:        "https://api.opsway.eu/v1/healthz",
					Assertions: []entities.MonitorAssertion{
						{
							Source:   "STATUS_CODE",
							Operator: "EQUAL",
							Target:   "200",
						},
					},
				},
			},
		},
		{
			Name:   "Dashboard",
			TeamID: t.ID,
			State:  entities.MonitorStateActive,
			Settings: entities.MonitorSettings{
				Frequency: time.Minute * 1,
			},
			Steps: []entities.MonitorStep{
				{
					OrderIndex: 0,
					Name:       "Ping",
					Method:     "GET",
					URL:        "https://my.opsway.eu",
					Assertions: []entities.MonitorAssertion{
						{
							Source:   "STATUS_CODE",
							Operator: "EQUAL",
							Target:   "200",
						},
					},
				},
			},
		},
		{
			Name:   "Website",
			TeamID: t.ID,
			State:  entities.MonitorStateActive,
			Settings: entities.MonitorSettings{
				Frequency: time.Minute * 1,
			},
			Steps: []entities.MonitorStep{
				{
					OrderIndex: 0,
					Name:       "Ping",
					Method:     "GET",
					URL:        "https://opsway.eu",
					Assertions: []entities.MonitorAssertion{
						{
							Source:   "STATUS_CODE",
							Operator: "EQUAL",
							Target:   "200",
						},
					},
				},
			},
		},
		{
			Name:   "Status Page",
			TeamID: t.ID,
			State:  entities.MonitorStateActive,
			Settings: entities.MonitorSettings{
				Frequency: time.Minute * 1,
			},
			Steps: []entities.MonitorStep{
				{
					OrderIndex: 0,
					Name:       "Ping",
					Method:     "GET",
					URL:        "https://status.opsway.eu",
					Assertions: []entities.MonitorAssertion{
						{
							Source:   "STATUS_CODE",
							Operator: "EQUAL",
							Target:   "200",
						},
					},
				},
			},
		},
	}

	var seededMonitors []entities.Monitor
	for _, m := range monitorsToSeed {
		var existing entities.Monitor
		err := db.Where("team_id = ? AND name = ?", m.TeamID, m.Name).First(&existing).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&m).Error; err != nil {
					panic(err)
				}
				seededMonitors = append(seededMonitors, m)
			} else {
				panic(err)
			}
		} else {
			existing.State = m.State
			db.Save(&existing)
			seededMonitors = append(seededMonitors, existing)
		}
	}

	// Status Page
	sp := entities.StatusPage{
		Domain: "status.opsway.eu",
	}
	if err := db.Where("domain = ?", sp.Domain).FirstOrCreate(&sp).Error; err != nil {
		panic(err)
	}
	sp.TeamID = t.ID
	sp.Name = "Opsway Status"
	sp.Layout = "STATS"
	sp.ShowBranding = true
	sp.IsPrivate = false
	if err := db.Save(&sp).Error; err != nil {
		panic(err)
	}

	if err := db.Model(&sp).Association("Monitors").Replace(seededMonitors); err != nil {
		panic(err)
	}
}
