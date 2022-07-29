package cmd

import (
	"context"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/jwt"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/rest"
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
		monitor.Monitor{},
		monitor.Settings{},
	)

	userService := user.NewService(db)

	// TODO: Remove
	u := &user.User{
		Name:        "Douglas Adams",
		DisplayName: "Ford Prefect",
		Email:       "admin@opsway.io",
	}
	u.SetPassword("pass")
	userService.CreateUser(ctx, u)
	// TODO: Remove

	jwtService := jwt.NewService(conf.JWT)

	srv, err := rest.NewServer(conf.REST, l, userService, jwtService)
	if err != nil {
		l.WithError(err).Fatal("Failed to create REST server")
	}

	if err := srv.Start(); err != nil {
		l.WithError(err).Fatal("Failed to start REST server")
	}
}
