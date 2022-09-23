package cmd

import (
	"context"

	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/rest"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/utils/pointer"
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
		entities.Team{},
		entities.User{},
		entities.Monitor{},
		entities.MonitorSettings{},
		entities.Maintenance{},
		entities.MaintenanceSettings{},
		entities.MaintenanceComment{},
		entities.Incident{},
		entities.IncidentComment{},
	)

	authenticationService := authentication.NewService(conf.Authentication)

	userRepository := user.NewRepository(db)
	userService := user.NewService(userRepository)

	teamRepository := team.NewRepository(db)
	teamService := team.NewService(teamRepository)

	monitorService := monitor.NewService(db)

	// TODO: Remove
	t := entities.Team{
		Name: "opsway",
	}
	db.Create(&t)

	u := &entities.User{
		Name:        "Douglas Adams",
		DisplayName: pointer.String("Ford Prefect"),
		Email:       "admin@opsway.io",
		Teams: []entities.Team{
			{
				ID: 1,
			},
		},
	}
	u.SetPassword("pass")

	// This creates the association to the team
	// but not the team itself.
	db.Omit("Teams.*").Create(u)
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
