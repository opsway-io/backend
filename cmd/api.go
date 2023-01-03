package cmd

import (
	"context"
	"time"

	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/rest"
	"github.com/opsway-io/backend/internal/storage"
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
		logrus.WithError(err).Fatal("Failed to load config")
	}

	l := getLogger(conf.Log)

	l.WithFields(logrus.Fields{
		"port": conf.REST.Port,
	}).Info("Starting REST server")

	ctx := context.Background()

	redisClient, err := redis.NewClient(ctx, conf.Redis)
	if err != nil {
		l.WithError(err).Fatal("Failed to connect to Redis")
	}

	db, err := postgres.NewClient(ctx, conf.Postgres)
	if err != nil {
		l.WithError(err).Fatal("Failed to create Postgres client")
	}

	db.SetupJoinTable(&entities.Team{}, "Users", &entities.TeamUser{})

	db.AutoMigrate(
		entities.User{},
		entities.Team{},
		entities.Monitor{},
		entities.MonitorSettings{},
		entities.Maintenance{},
		entities.MaintenanceSettings{},
		entities.MaintenanceComment{},
		entities.Incident{},
		entities.IncidentComment{},
	)

	// CLICKHOUSE
	ch_db, err := clickhouse.NewClient(ctx, conf.Clickhouse)
	if err != nil {
		l.WithError(err).Fatal("Failed to create clickhouse")
	}

	ch_db.AutoMigrate(
		check.Check{},
	)

	storageRepository := storage.NewObjectStorageRepository(ctx, conf.ObjectStorage)
	storageService := storage.NewService(storageRepository)

	authenticationService := authentication.NewService(conf.Authentication, redisClient)

	userRepository := user.NewRepository(db)
	userService := user.NewService(userRepository, storageService)

	teamRepository := team.NewRepository(db)
	teamService := team.NewService(teamRepository, storageService)

	monitorService := monitor.NewService(db, redisClient)

	httpResultService := check.NewService(ch_db)

	// TODO: Remove

	u := &entities.User{
		Name:        "Douglas Adams",
		DisplayName: pointer.String("Ford Prefect"),
		Email:       "admin@opsway.io",
	}
	u.SetPassword("pass")
	db.FirstOrCreate(u)

	t := entities.Team{
		Name: "opsway",
	}
	db.FirstOrCreate(&t)

	db.FirstOrCreate(&entities.TeamUser{
		UserID: u.ID,
		TeamID: t.ID,
		Role:   entities.TeamRoleOwner,
	})

	m := &entities.Monitor{
		Name: "opsway.io",
		Settings: entities.MonitorSettings{
			Method:    "GET",
			URL:       "https://opsway.io",
			Frequency: time.Second * 10,
		},
		TeamID: t.ID,
	}

	db.FirstOrCreate(m)

	// TODO: Remove

	srv, err := rest.NewServer(
		conf.REST,
		conf.OAuth,
		l,
		authenticationService,
		userService,
		teamService,
		monitorService,
		httpResultService,
	)
	if err != nil {
		l.WithError(err).Fatal("Failed to create REST server")
	}

	if err := srv.Start(); err != nil {
		l.WithError(err).Fatal("Failed to start REST server")
	}
}
