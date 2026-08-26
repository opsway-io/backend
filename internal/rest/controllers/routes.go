package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/apikey"
	auth "github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/billing"
	"github.com/opsway-io/backend/internal/changelog"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/escalation"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/heartbeats"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/maintenance"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/report"
	alertingController "github.com/opsway-io/backend/internal/rest/controllers/alerting"
	"github.com/opsway-io/backend/internal/rest/controllers/apikeys"
	"github.com/opsway-io/backend/internal/rest/controllers/authentication"
	"github.com/opsway-io/backend/internal/rest/controllers/changelogs"
	"github.com/opsway-io/backend/internal/rest/controllers/healthz"
	heartbeatsController "github.com/opsway-io/backend/internal/rest/controllers/heartbeats"
	"github.com/opsway-io/backend/internal/rest/controllers/incidents"
	maintenanceController "github.com/opsway-io/backend/internal/rest/controllers/maintenance"
	"github.com/opsway-io/backend/internal/rest/controllers/metrics"
	"github.com/opsway-io/backend/internal/rest/controllers/monitors"
	"github.com/opsway-io/backend/internal/rest/controllers/prober"
	"github.com/opsway-io/backend/internal/rest/controllers/prometheus"
	"github.com/opsway-io/backend/internal/rest/controllers/reports"
	"github.com/opsway-io/backend/internal/rest/controllers/statuspages"
	"github.com/opsway-io/backend/internal/rest/controllers/teams"
	"github.com/opsway-io/backend/internal/rest/controllers/users"
	"github.com/opsway-io/backend/internal/rest/controllers/webhooks"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Register(
	e *echo.Echo,
	logger *logrus.Entry,
	oAuthConfig *authentication.OAuthConfig,
	authConfig *auth.Config,
	cookieService helpers.CookieService,
	authenticationService auth.Service,
	userService user.Service,
	teamService team.Service,
	alertingService alerting.Service,
	monitorService monitor.Service,
	checkService check.Service,
	billingService billing.Service,
	changelogService changelog.Service,
	heartbeatService heartbeats.Service,
	incidentService incident.Service,
	maintenanceService maintenance.Service,
	reportsService report.Service,
	statusPageService statuspage.Service,
	escalationService escalation.Service,
	eventService event.Service,
	apiKeyService apikey.Service,
	emailSender email.Sender,
	availableLocations []string,
	db *gorm.DB,
	ch *gorm.DB,
	statusPageBaseURL string,
) {
	AuthGuard := middleware.AuthGuardFactory(logger, authenticationService)

	root := e.Group(
		"/v1",
	)

	authRoot := root.Group(
		"",
		AuthGuard(),
	)

	// Webhooks

	webhooks.Register(
		root,
		logger.WithField("module", "webhooks"),
		billingService,
		teamService,
		incidentService,
		emailSender,
	)

	// Healthz

	healthz.Register(root, logger)

	// Authentication

	authentication.Register(root, logger, cookieService, oAuthConfig, authConfig, authenticationService, teamService, userService)

	// Users

	users.Register(authRoot, logger, teamService, userService)

	// Teams

	teams.Register(authRoot, logger, teamService, userService, billingService, escalationService)

	// API Keys

	apikeys.Register(authRoot, logger, teamService, apiKeyService)

	// Monitors

	monitors.Register(authRoot, logger, teamService, monitorService, checkService, maintenanceService)

	// Changelogs

	changelogs.Register(authRoot, logger, teamService, changelogService)

	// Alerting
	alertingController.Register(authRoot, logger, teamService, alertingService)

	// Incidents
	incidents.Register(authRoot, logger, teamService, incidentService, eventService)

	// Heartbeats
	heartbeatsController.Register(authRoot, logger, teamService, heartbeatService)

	// Maintenance
	maintenanceController.Register(authRoot, logger, teamService, maintenanceService, monitorService)

	// Reports
	reports.Register(authRoot, logger, teamService, reportsService, checkService, eventService)

	statuspages.Register(authRoot, logger, teamService, statusPageService)
	statuspages.RegisterPublic(root, logger, statusPageBaseURL, statusPageService, incidentService, maintenanceService, emailSender)

	// Prober
	prober.Register(root, logger, availableLocations)

	// Prometheus Metrics
	prometheus.Register(e, logger, db, ch)

	// API Key Metrics
	metrics.Register(root, logger, checkService, apiKeyService, db)
}
