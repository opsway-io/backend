package cmd

import (
	"context"
	"time"

	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/expiry"
	"github.com/opsway-io/backend/internal/forecaster"
	"github.com/opsway-io/backend/internal/maintenance"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// MaintainerCmd starts the maintenance worker
var MaintainerCmd = &cobra.Command{
	Use:   "maintainer",
	Short: "Start the maintenance worker",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		conf, err := loadConfig()
		if err != nil {
			logrus.WithError(err).Fatal("Failed to load config")
		}

		logger := getLogger(conf.Log)

		logger.Info("Starting maintainer")

		db, err := postgres.NewClient(ctx, conf.Postgres)
		if err != nil {
			logger.WithError(err).Fatal("failed to connect to postgres")
		}

		redisClient, err := redis.NewClient(ctx, conf.Redis)
		if err != nil {
			logger.WithError(err).Fatal("failed to connect to redis")
		}

		monitorService := monitor.NewService(db, redisClient)
		eventService, err := event.NewService(redisClient, "maintainer")
		if err != nil {
			logger.WithError(err).Fatal("failed to create event service")
		}

		maintenanceRepo := maintenance.NewRepository(db)
		maintenanceService := maintenance.NewService(maintenanceRepo, eventService)

		statuspageRepo := statuspage.NewRepository(db)
		statuspageService := statuspage.NewService(statuspageRepo, nil)

		var emailSender email.Sender
		if conf.Email.Debug {
			logger.Info("Using SMTP email sender")
			emailSender = email.NewSMTPSender(conf.Email)
		} else {
			logger.Info("Using Sendgrid email sender")
			emailSender = email.NewSendgridSender(conf.Email)
		}

		// Poll every 10 seconds (configurable if needed)
		worker := maintenance.NewWorker(logger.WithField("module", "maintainer"), maintenanceService, monitorService, statuspageService, emailSender, 10*time.Second, conf.StatusPage.BaseURL)
		go func() {
			if err := worker.Start(ctx); err != nil {
				logger.WithError(err).Fatal("maintenance worker failed")
			}
		}()

		// Train models daily (every 24 hours)
		forecasterWorker := forecaster.NewWorker(monitorService, eventService, logger.WithField("module", "forecaster_trainer"), 24*time.Hour)
		go func() {
			if err := forecasterWorker.Start(ctx); err != nil {
				logger.WithError(err).Fatal("forecaster worker failed")
			}
		}()

		// Start Expiry Worker (runs once per hour)
		chDB, err := clickhouse.NewClient(ctx, conf.Clickhouse)
		if err != nil {
			logger.WithError(err).Fatal("failed to connect to clickhouse")
		}

		checkRepo := check.NewRepository(chDB)
		checkSvc := check.NewService(checkRepo)
		
		teamRepo := team.NewRepository(db)
		teamCache := team.NewCache(redisClient)
		teamSvc := team.NewService(conf.Team, teamRepo, nil, emailSender, teamCache) // nil storage is fine

		expiryWorker := expiry.NewWorker(logger.WithField("module", "expiry_worker"), monitorService, checkSvc, teamSvc, emailSender, time.Hour, conf.Team.ApplicationURL)
		
		if err := expiryWorker.Start(ctx); err != nil {
			logger.WithError(err).Fatal("expiry worker failed")
		}
	},
}

func init() {
	rootCmd.AddCommand(MaintainerCmd)
}
