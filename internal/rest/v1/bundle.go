package v1

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/jwt"
	"github.com/opsway-io/backend/internal/rest/v1/authentication"
	"github.com/opsway-io/backend/internal/user"
	"go.uber.org/zap"
)

func Register(e *echo.Group, logger *zap.Logger, userService user.Service, jwtService jwt.Service) {
	g := e.Group("/v1")

	authentication.Register(g, logger, userService, jwtService)
}
