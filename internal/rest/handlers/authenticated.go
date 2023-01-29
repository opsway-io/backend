package handlers

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/sirupsen/logrus"
)

type AuthenticatedHandlerFunc func(ctx AuthenticatedContext) error

type AuthenticatedContext struct {
	echo.Context
	Log    *logrus.Entry
	Claims authentication.Claims
	UserID uint
}

func AuthenticatedHandlerFactory(logger *logrus.Entry) func(handler AuthenticatedHandlerFunc) func(ctx echo.Context) error {
	return func(handler AuthenticatedHandlerFunc) func(ctx echo.Context) error {
		return func(ctx echo.Context) error {
			claims, ok := ctx.Get("jwt_claims").(*authentication.Claims)
			if !ok {
				logger.Debug("missing jwt_claims")

				return echo.ErrUnauthorized
			}

			userId, err := strconv.ParseUint(claims.Subject, 10, 64)
			if err != nil {
				logger.WithError(err).Debug("failed to parse subject to user id")

				return echo.ErrUnauthorized
			}

			return handler(AuthenticatedContext{
				Context: ctx,
				Claims:  *claims,
				Log:     logger,
				UserID:  uint(userId),
			})
		}
	}
}
