package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/check"
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
	CheckService          check.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	authenticationService authentication.Service,
	userService user.Service,
	teamService team.Service,
	monitorService monitor.Service,
	checkService check.Service,
) {
	h := &Handlers{
		AuthenticationService: authenticationService,
		UserService:           userService,
		TeamService:           teamService,
		MonitorService:        monitorService,
		CheckService:          checkService,
	}

	AuthGuard := mw.AuthGuardFactory(logger, authenticationService)
	CurrentUserGuard := mw.CurrentUSerGuardFactory(logger)
	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)

	BaseHandler := handlers.BaseHandlerFactory(logger)
	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	// Authentication

	authGroup := e.Group("/auth")

	authGroup.POST("/login", BaseHandler(h.PostLogin))
	authGroup.POST("/refresh", BaseHandler(h.PostRefreshToken))

	// Users

	usersGroup := e.Group(
		"/users/:userId",
		AuthGuard(),
		CurrentUserGuard(),
	)

	usersGroup.GET("", AuthHandler(h.GetUser))
	usersGroup.PUT("", AuthHandler(h.PutUser))
	usersGroup.DELETE("", AuthHandler(h.DeleteUser))
	usersGroup.PUT("/password", AuthHandler(h.PutUserPassword))
	usersGroup.PUT("/avatar", AuthHandler(h.PutUserAvatar))
	usersGroup.DELETE("/avatar", AuthHandler(h.DeleteUserAvatar))

	// Teams

	teamsGroup := e.Group(
		"/teams/:teamId",
		AuthGuard(),
		TeamGuard(),
	)

	teamsGroup.GET("", AuthHandler(h.GetTeam))
	teamsGroup.PUT("", AuthHandler(h.PutTeam))
	teamsGroup.GET("/users", AuthHandler(h.GetTeamUsers))

	// Monitors

	monitorsGroup := e.Group(
		"/teams/:teamId/monitors",
		AuthGuard(),
		TeamGuard(),
	)

	monitorsGroup.GET("", AuthHandler(h.GetMonitors))
	monitorsGroup.POST("", AuthHandler(h.PostMonitor), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	monitorsGroup.GET("/:monitorId", AuthHandler(h.GetMonitor))
	monitorsGroup.DELETE("/:monitorId", AuthHandler(h.DeleteMonitor), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	monitorsGroup.GET("/:monitorId/checks", AuthHandler(h.GetMonitorChecks))
	monitorsGroup.GET("/:monitorId/metrics", AuthHandler(h.GetMonitorMetrics))
}
