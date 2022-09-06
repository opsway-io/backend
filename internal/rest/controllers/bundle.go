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

	// Authentication

	authGroup := e.Group("/authentication")

	authGroup.POST("/login", hs.StandardHandler(h.PostLogin, logger))

	// Users

	usersGroup := e.Group(
		"/users/:userId",
		middleware.AuthGuard(logger, authenticationService),
	)

	usersGroup.GET("", hs.AuthenticatedHandler(h.GetUser, logger))
	usersGroup.PUT("", hs.AuthenticatedHandler(h.PutUser, logger))

	// Teams

	teamsGroup := e.Group(
		"/teams/:teamId",
		middleware.AuthGuard(logger, authenticationService),
		middleware.TeamGuard(logger),
	)

	teamsGroup.GET("", hs.AuthenticatedHandler(h.GetTeam, logger))
	teamsGroup.PUT("", hs.AuthenticatedHandler(h.PutTeam, logger))
	teamsGroup.GET("/users", hs.AuthenticatedHandler(h.GetTeamUsers, logger))

	// Monitors

	monitorsGroup := e.Group(
		"/teams/:team_id/monitors",
		middleware.AuthGuard(logger, authenticationService),
		middleware.TeamGuard(logger),
	)

	monitorsGroup.GET("", hs.AuthenticatedHandler(h.GetMonitors, logger))
	monitorsGroup.GET("/:monitor_id", hs.AuthenticatedHandler(h.GetMonitor, logger))
	monitorsGroup.POST("", hs.AuthenticatedHandler(h.PostMonitor, logger))
	monitorsGroup.PUT("/:monitor_id", hs.AuthenticatedHandler(h.PutMonitor, logger))
	monitorsGroup.DELETE("/:monitor_id", hs.AuthenticatedHandler(h.DeleteMonitor, logger))
}
