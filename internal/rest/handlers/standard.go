package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/jwt"
	"github.com/sirupsen/logrus"
)

type StandardHandlerFunc func(ctx echo.Context, logger *logrus.Entry) error

func StandardHandler(handler StandardHandlerFunc, logger *logrus.Entry) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return handler(ctx, logger)
	}
}

type AuthenticatedContext struct {
	echo.Context
	Claims jwt.Claims
}
