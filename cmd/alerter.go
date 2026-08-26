package cmd

import (
	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	connectorRedis "github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/escalation"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/llm"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/opsway-io/backend/internal/storage"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var alerterCmd = &cobra.Command{
	Use: "alerter",
	Run: runAlerter,
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(alerterCmd)
}

func runAlerter(cmd *cobra.Command, args []string) {
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

	eventService, err := event.NewService(redisClient, "alerter")
	if err != nil {
		l.WithError(err).Fatal("Failed to create event service")
	}

	var emailSender email.Sender
	if conf.Email.Debug {
		l.Info("Using SMTP email sender")
		emailSender = email.NewSMTPSender(conf.Email)
	} else {
		l.Info("Using Sendgrid email sender")
		emailSender = email.NewSendgridSender(conf.Email)
	}

	alertingRepository := alerting.NewRepository(db)
	alertingService := alerting.NewService(alertingRepository)

	storageRepository := storage.NewObjectStorageRepository(ctx, conf.ObjectStorage)
	storageService := storage.NewService(storageRepository)

	teamRepository := team.NewRepository(db)
	teamCache := team.NewCache(redisClient)
	teamService := team.NewService(conf.Team, teamRepository, storageService, emailSender, teamCache)

	monitorService := monitor.NewService(db, redisClient)

	statuspageRepo := statuspage.NewRepository(db)
	statuspageService := statuspage.NewService(statuspageRepo, nil) // nil k8sService is fine since alerting worker doesn't modify ingresses

	workerConfig := alerting.WorkerConfig{
		ApplicationURL:    conf.Team.ApplicationURL,
		StatusPageBaseURL: conf.StatusPage.BaseURL,
	}

	escalationRepo := escalation.NewRepository(db)
	escalationSvc := escalation.NewService(escalationRepo)

	incidentRepo := incident.NewRepository(db)
	incidentSvc := incident.NewService(incidentRepo, eventService)

	worker := alerting.NewWorker(
		workerConfig,
		eventService,
		alertingService,
		teamService,
		monitorService,
		statuspageService,
		emailSender,
		escalationSvc,
		incidentSvc,
		l.WithField("module", "alerter"),
	)

	llmClient := llm.NewClient("", "", "") // Use defaults for MVP
	rcaWorker := incident.NewRCAWorker(
		eventService,
		incidentSvc,
		llmClient,
		l.WithField("module", "rca_worker"),
	)

	go func() {
		if err := rcaWorker.Start(ctx); err != nil {
			l.WithError(err).Error("failed to start rca worker")
		}
	}()

	l.Info("Alerter daemon is listening for incidents...")
	if err := worker.Start(ctx); err != nil {
		l.WithError(err).Fatal("Alerter worker exited with error")
	}
}
