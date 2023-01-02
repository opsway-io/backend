package controllers

import (
	"github.com/labstack/echo/v4"
	auth "github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/rest/controllers/authentication"
	"github.com/opsway-io/backend/internal/rest/controllers/monitors"
	"github.com/opsway-io/backend/internal/rest/controllers/teams"
	"github.com/opsway-io/backend/internal/rest/controllers/users"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	authenticationService auth.Service,
	userService user.Service,
	teamService team.Service,
	monitorService monitor.Service,
	checkService check.Service,
) {
	// Authentication

	authentication.Register(e, logger, authenticationService, teamService, userService)

	// Users

	users.Register(e, logger, authenticationService, teamService, userService)

	// Teams

	teams.Register(e, logger, authenticationService, teamService, userService)

	// Monitors

	monitors.Register(e, logger, authenticationService, teamService, monitorService, checkService)
}
