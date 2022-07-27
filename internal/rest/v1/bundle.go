package v1

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/jwt"
	"github.com/opsway-io/backend/internal/rest/v1/authentication"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

func Register(e *echo.Group, logger *logrus.Logger, userService user.Service, jwtService jwt.Service) {
	l := logger.WithFields(logrus.Fields{})

	g := e.Group("/v1")

	authentication.Register(g, l, userService, jwtService)
}
