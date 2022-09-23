package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
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

	AuthGuard := mw.AuthGuardFactory(logger, authenticationService)
	TeamGuard := mw.TeamGuardFactory(logger)
	RoleGuard := mw.RoleGuardFactory(logger)

	BaseHandler := handlers.BaseHandlerFactory(logger)
	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	// Authentication

	authGroup := e.Group("/authentication")

	authGroup.POST("/login", BaseHandler(h.PostLogin))

	// Users

	usersGroup := e.Group(
		"/users/:userId",
		AuthGuard(),
	)

	usersGroup.GET("", AuthHandler(h.GetUser))
	usersGroup.PUT("", AuthHandler(h.PutUser))

	// Teams

	teamsGroup := e.Group(
		"/teams/:teamId",
		AuthGuard(),
		TeamGuard(),
	)

	teamsGroup.GET("", AuthHandler(h.GetTeam))
	teamsGroup.PUT("", AuthHandler(h.PutTeam), RoleGuard(mw.UserRoleAdmin))

	teamsGroup.GET("/users", AuthHandler(h.GetTeamUsers))

	// Monitors

	monitorsGroup := e.Group(
		"/teams/:teamId/monitors",
		AuthGuard(),
		TeamGuard(),
	)

	monitorsGroup.GET("", AuthHandler(h.GetMonitors))
	monitorsGroup.POST("", AuthHandler(h.PostMonitor), RoleGuard(mw.UserRoleAdmin))

	monitorsGroup.GET("/:monitor_id", AuthHandler(h.GetMonitor))
	monitorsGroup.PUT("/:monitor_id", AuthHandler(h.PutMonitor), RoleGuard(mw.UserRoleAdmin))
	monitorsGroup.DELETE("/:monitor_id", AuthHandler(h.DeleteMonitor), RoleGuard(mw.UserRoleAdmin))
}
