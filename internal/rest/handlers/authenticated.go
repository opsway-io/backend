package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/sirupsen/logrus"
)

type AuthenticatedHandlerFunc func(ctx AuthenticatedContext, l *logrus.Entry) error

type AuthenticatedContext struct {
	echo.Context
	Claims authentication.Claims
}

func AuthenticatedHandler(handler AuthenticatedHandlerFunc, logger *logrus.Entry) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		claims, ok := ctx.Get("jwt_claims").(*authentication.Claims)
		if !ok {
			logger.Debug("missing jwt_claims")

			return echo.ErrUnauthorized
		}

		return handler(AuthenticatedContext{
			Context: ctx,
			Claims:  *claims,
		}, logger)
	}
}
