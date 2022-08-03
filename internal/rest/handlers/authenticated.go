package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/jwt"
	"github.com/sirupsen/logrus"
)

type AuthenticatedHandlerFunc func(ctx echo.Context, l *logrus.Entry) error

func AuthenticatedHandler(handler AuthenticatedHandlerFunc, logger *logrus.Entry) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		claims, ok := ctx.Get("jwt_claims").(*jwt.Claims)
		if !ok {
			logger.Debug("missing jwt_claims")

			return echo.ErrUnauthorized
		}

		return handler(&AuthenticatedContext{
			Context: ctx,
			Claims:  *claims,
		}, logger)
	}
}
