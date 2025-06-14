package monitors

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	TeamService           team.Service
	CheckService          check.Service
	MonitorService        monitor.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	monitorService monitor.Service,
	checkService check.Service,
) {
	h := &Handlers{
		MonitorService: monitorService,
		CheckService:   checkService,
	}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	monitorsGroup := e.Group(
		"/teams/:teamId/monitors",
		TeamGuard(),
	)

	monitorsGroup.GET("", AuthHandler(h.GetMonitors))
	monitorsGroup.POST("", AuthHandler(h.PostMonitor), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	monitorsGroup.GET("/incidents", AuthHandler(h.GetMonitorIncidents))

	monitorsGroup.GET("/:monitorId", AuthHandler(h.GetMonitor))
	monitorsGroup.DELETE("/:monitorId", AuthHandler(h.DeleteMonitor), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	monitorsGroup.PUT("/:monitorId", AuthHandler(h.PutMonitor), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	monitorsGroup.PUT("/:monitorId/state", AuthHandler(h.PutMonitorState), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	monitorsGroup.GET("/:monitorId/checks", AuthHandler(h.GetMonitorChecks))
	monitorsGroup.GET("/:monitorId/checks/failed/:monitorAssertionId", AuthHandler(h.GetFailedMonitorChecks))
	monitorsGroup.GET("/:monitorId/checks/:checkId", AuthHandler(h.GetMonitorCheck))

	monitorsGroup.GET("/:monitorId/metrics", AuthHandler(h.GetMonitorMetrics))
}
