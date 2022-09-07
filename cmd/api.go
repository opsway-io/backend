package cmd

import (
	"context"

	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/maintenance"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/rest"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var apiCmd = &cobra.Command{
	Use: "api",
	Run: runAPI,
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(apiCmd)
}

func runAPI(cmd *cobra.Command, args []string) {
	conf, err := loadConfig()
	if err != nil {
		panic(err)
	}

	l := getLogger(conf.Log)

	l.WithFields(logrus.Fields{
		"port": conf.REST.Port,
	}).Info("Starting REST server")

	ctx := context.Background()

	db, err := postgres.NewClient(ctx, conf.Postgres)
	if err != nil {
		l.WithError(err).Fatal("Failed to create Postgres client")
	}

	db.AutoMigrate(
		user.User{},
		team.Team{},
		monitor.Monitor{},
		monitor.Settings{},
		maintenance.Maintenance{},
		maintenance.Settings{},
		maintenance.Comment{},
	)

	authenticationService := authentication.NewService(conf.Authentication)

	userRepository := user.NewRepository(db)
	userService := user.NewService(userRepository)

	teamRepository := team.NewRepository(db)
	teamService := team.NewService(teamRepository)

	monitorService := monitor.NewService(db)

	// TODO: Remove
	// t := team.Team{
	// 	Name: "opsway",
	// }
	// db.Create(&t)

	// u := &user.User{
	// 	Name:        "Douglas Adams",
	// 	DisplayName: "Ford Prefect",
	// 	Email:       "admin@opsway.io",
	// 	TeamID:      t.ID,
	// }
	// u.SetPassword("pass")
	// db.Create(u)

	// m := maintenance.Maintenance{
	// 	Title:  "Test",
	// 	TeamID: 1,
	// 	Settings: maintenance.Settings{
	// 		StartAt: time.Now(),
	// 		EndAt:   time.Now().Add(1 * time.Hour),
	// 	},
	// }
	// db.Create(&m)

	// c := maintenance.Comment{
	// 	Content:       "Test",
	// 	UserID:        1,
	// 	MaintenanceID: 1,
	// }
	// db.Create(&c)

	// TODO: Remove

	srv, err := rest.NewServer(
		conf.REST,
		l,
		authenticationService,
		userService,
		teamService,
		monitorService,
	)
	if err != nil {
		l.WithError(err).Fatal("Failed to create REST server")
	}

	if err := srv.Start(); err != nil {
		l.WithError(err).Fatal("Failed to start REST server")
	}
}
