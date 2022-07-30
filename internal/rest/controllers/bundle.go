package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/jwt"
	"github.com/opsway-io/backend/internal/monitor"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	UserService    user.Service
	JWTService     jwt.Service
	MonitorService monitor.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	userService user.Service,
	jwtService jwt.Service,
	monitorService monitor.Service,
) {
	h := &Handlers{
		UserService:    userService,
		JWTService:     jwtService,
		MonitorService: monitorService,
	}

	// Authentication

	authGroup := e.Group("/authentication")

	authGroup.POST("/login", hs.StandardHandler(h.PostLogin, logger))
	authGroup.POST("/refresh", hs.StandardHandler(h.PostRefresh, logger))

	// Monitors

	monitorsGroup := e.Group(
		"/teams/:team_id/monitors",
		middleware.AuthGuard(logger, jwtService),
		middleware.TeamGuard(logger),
	)

	monitorsGroup.GET("", hs.AuthenticatedHandler(h.GetMonitors, logger))
	monitorsGroup.GET("/:monitor_id", hs.AuthenticatedHandler(h.GetMonitor, logger))
}
