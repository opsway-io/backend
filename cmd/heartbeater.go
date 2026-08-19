package cmd

import (
	"time"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	connectorRedis "github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/heartbeats"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var heartbeaterCmd = &cobra.Command{
	Use: "heartbeater",
	Run: runHeartbeater,
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(heartbeaterCmd)
}

func runHeartbeater(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	conf, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	l := getLogger(conf.Log)

	l.WithFields(logrus.Fields{
		"host": conf.Redis.Host,
		"port": conf.Redis.Port,
		"db":   conf.Redis.DB,
	}).Info("Connecting to redis")

	redisClient, err := connectorRedis.NewClient(ctx, conf.Redis)
	if err != nil {
		l.WithError(err).Fatal("failed to connect to redis")
	}

	db, err := postgres.NewClient(ctx, conf.Postgres)
	if err != nil {
		l.WithError(err).Fatal("Failed to create Postgres client")
	}

	eventService, err := event.NewService(redisClient)
	if err != nil {
		l.WithError(err).Fatal("Failed to create event service")
	}

	heartbeatRepository := heartbeats.NewRepository(db)
	heartbeatService := heartbeats.NewService(heartbeatRepository)

	incidentRepository := incident.NewRepository(db)
	incidentService := incident.NewService(incidentRepository, eventService)

	worker := heartbeats.NewWorker(
		heartbeatService,
		incidentService,
		l.WithField("module", "heartbeater"),
		time.Second*10,
	)

	if err := worker.Start(ctx); err != nil {
		l.WithError(err).Fatal("Heartbeater worker exited with error")
	}
}
