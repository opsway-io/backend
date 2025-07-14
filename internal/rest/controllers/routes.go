package controllers

import (
	"github.com/labstack/echo/v4"
	auth "github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/billing"
	"github.com/opsway-io/backend/internal/changelog"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/report"
	"github.com/opsway-io/backend/internal/rest/controllers/authentication"
	"github.com/opsway-io/backend/internal/rest/controllers/changelogs"
	"github.com/opsway-io/backend/internal/rest/controllers/healthz"
	"github.com/opsway-io/backend/internal/rest/controllers/incidents"
	"github.com/opsway-io/backend/internal/rest/controllers/monitors"
	"github.com/opsway-io/backend/internal/rest/controllers/reports"
	"github.com/opsway-io/backend/internal/rest/controllers/teams"
	"github.com/opsway-io/backend/internal/rest/controllers/users"
	"github.com/opsway-io/backend/internal/rest/controllers/webhooks"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
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
	monitorService monitor.Service,
	checkService check.Service,
	billingService billing.Service,
	changelogService changelog.Service,
	incidentService incident.Service,
	reportsService report.Service,
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

	webhooks.Register(root, logger, billingService, teamService)

	// Healthz

	healthz.Register(root, logger)

	// Authentication

	authentication.Register(root, logger, cookieService, oAuthConfig, authConfig, authenticationService, teamService, userService)

	// Users

	users.Register(authRoot, logger, teamService, userService)

	// Teams

	teams.Register(authRoot, logger, teamService, userService, billingService)

	// Monitors

	monitors.Register(authRoot, logger, teamService, monitorService, checkService)

	// Changelogs

	changelogs.Register(authRoot, logger, teamService, changelogService)

	// Incidents
	incidents.Register(authRoot, logger, teamService, incidentService)

	// Reports
	reports.Register(authRoot, logger, teamService, reportsService, checkService)
}
