package cmd

import (
	"context"

	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/billing"
	"github.com/opsway-io/backend/internal/changelog"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/heartbeats"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/k8s"
	"github.com/opsway-io/backend/internal/maintenance"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/report"
	"github.com/opsway-io/backend/internal/rest"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/opsway-io/backend/internal/apikey"
	"github.com/opsway-io/backend/internal/storage"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/escalation"
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
	db.SetupJoinTable(&entities.ChangelogEntry{}, "Authors", &entities.ChangelogEntryAuthor{})

	db.AutoMigrate(
		entities.User{},
		entities.Team{},
		entities.Monitor{},
		entities.MonitorSettings{},
		entities.MonitorAssertion{},
		entities.AlertRule{},
		entities.Maintenance{},
		entities.MaintenanceSettings{},
		entities.MaintenanceComment{},
		entities.Incident{},
		entities.IncidentComment{},
		entities.Changelog{},
		entities.ChangelogEntry{},
		entities.Report{},
		entities.StatusPage{},
		entities.StatusPageSubscriber{},
		entities.APIKey{},
		entities.EscalationPolicy{},
		entities.OnCallRotation{},
		entities.TeamInvitation{},
	)

	ch_db, err := clickhouse.NewClient(ctx, conf.Clickhouse)
	if err != nil {
		l.WithError(err).Fatal("Failed to create clickhouse")
	}

	ch_db.AutoMigrate(
		check.Check{},
	)

	var emailSender email.Sender
	if conf.Email.Debug {
		l.Info("Using SMTP email sender")

		emailSender = email.NewSMTPSender(conf.Email)
	} else {
		l.Info("Using Sendgrid email sender")

		emailSender = email.NewSendgridSender(conf.Email)
	}

	eventService, err := event.NewService(redisClient)
	if err != nil {
		l.WithError(err).Fatal("Failed to create event service")
	}

	storageRepository := storage.NewObjectStorageRepository(ctx, conf.ObjectStorage)
	storageService := storage.NewService(storageRepository)

	authenticationService := authentication.NewService(conf.Authentication, redisClient)

	userRepository := user.NewRepository(db)
	userCache := user.NewCache(redisClient)
	userService := user.NewService(userRepository, userCache, storageService, emailSender, eventService, conf.User)

	teamRepository := team.NewRepository(db)
	teamCache := team.NewCache(redisClient)
	teamService := team.NewService(conf.Team, teamRepository, storageService, emailSender, teamCache)

	monitorService := monitor.NewService(db, redisClient)

	httpResultService := check.NewService(ch_db)

	alertingRepository := alerting.NewRepository(db)
	alertingService := alerting.NewService(alertingRepository)

	heartbeatsRepository := heartbeats.NewRepository(db)
	heartbeatsService := heartbeats.NewService(heartbeatsRepository)

	maintenanceRepository := maintenance.NewRepository(db)
	maintenanceService := maintenance.NewService(maintenanceRepository, eventService)

	k8sService := k8s.NewService(l)

	statuspageRepository := statuspage.NewRepository(db)
	statuspageService := statuspage.NewService(statuspageRepository, k8sService)

	billingService := billing.NewService(conf.Stripe)

	changelogService := changelog.NewService(db)

	incidentRepository := incident.NewRepository(db)
	incidentService := incident.NewService(incidentRepository, eventService)

	reportRepository := report.NewRepository(db)
	reportsService := report.NewService(reportRepository)

	apiKeyRepository := apikey.NewRepository(db)
	apiKeyService := apikey.NewService(apiKeyRepository)

	escalationRepo := escalation.NewRepository(db)
	escalationService := escalation.NewService(escalationRepo)

	srv, err := rest.NewServer(
		conf.REST,
		conf.OAuth,
		&conf.Authentication,
		l,
		authenticationService,
		userService,
		teamService,
		alertingService,
		monitorService,
		httpResultService,
		billingService,
		changelogService,
		heartbeatsService,
		incidentService,
		maintenanceService,
		reportsService,
		statuspageService,
		escalationService,
		eventService,
		apiKeyService,
		emailSender,
		conf.Prober.AvailableLocations,
		db,
		ch_db,
		conf.StatusPage.BaseURL,
	)
	if err != nil {
		l.WithError(err).Fatal("Failed to create REST server")
	}

	if err := srv.Start(); err != nil {
		l.WithError(err).Fatal("Failed to start REST server")
	}
}
