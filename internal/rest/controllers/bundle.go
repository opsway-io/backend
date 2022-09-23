package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/monitor"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	UserService           user.Service
	TeamService           team.Service
	MonitorService        monitor.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	authenticationService authentication.Service,
	userService user.Service,
	teamService team.Service,
	monitorService monitor.Service,
) {
	h := &Handlers{
		AuthenticationService: authenticationService,
		UserService:           userService,
		TeamService:           teamService,
		MonitorService:        monitorService,
	}

	AuthGuard := middleware.AuthGuardFactory(logger, authenticationService)
	TeamGuard := middleware.TeamGuardFactory(logger)

	// Authentication

	authGroup := e.Group("/authentication")

	authGroup.POST("/login", hs.BaseHandler(h.PostLogin, logger))

	// Users

	usersGroup := e.Group(
		"/users/:userId",
		AuthGuard(),
	)

	usersGroup.GET("", hs.AuthenticatedHandler(h.GetUser, logger))
	usersGroup.PUT("", hs.AuthenticatedHandler(h.PutUser, logger))

	// Teams

	teamsGroup := e.Group(
		"/teams/:teamId",
		AuthGuard(),
		TeamGuard(),
	)

	teamsGroup.GET("", hs.AuthenticatedHandler(h.GetTeam, logger))
	teamsGroup.PUT("", hs.AuthenticatedHandler(h.PutTeam, logger))
	teamsGroup.GET("/users", hs.AuthenticatedHandler(h.GetTeamUsers, logger))

	// Monitors

	monitorsGroup := teamsGroup.Group(
		"/monitors",
	)

	monitorsGroup.GET("", hs.AuthenticatedHandler(h.GetMonitors, logger))
	monitorsGroup.GET("/:monitor_id", hs.AuthenticatedHandler(h.GetMonitor, logger))
	monitorsGroup.POST("", hs.AuthenticatedHandler(h.PostMonitor, logger))
	monitorsGroup.PUT("/:monitor_id", hs.AuthenticatedHandler(h.PutMonitor, logger))
	monitorsGroup.DELETE("/:monitor_id", hs.AuthenticatedHandler(h.DeleteMonitor, logger))
}
