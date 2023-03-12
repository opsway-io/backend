package handlers

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/sirupsen/logrus"
)

type AuthenticatedHandlerFunc func(c AuthenticatedContext) error

type AuthenticatedContext struct {
	echo.Context
	Log      *logrus.Entry
	Claims   authentication.Claims
	UserID   uint
	TeamRole *entities.TeamRole
}

func AuthenticatedHandlerFactory(logger *logrus.Entry) func(handler AuthenticatedHandlerFunc) func(c echo.Context) error {
	return func(handler AuthenticatedHandlerFunc) func(c echo.Context) error {
		return func(c echo.Context) error {
			claims, ok := c.Get("jwt_claims").(*authentication.Claims)
			if !ok {
				logger.Debug("missing jwt_claims")

				return echo.ErrUnauthorized
			}

			userId, err := strconv.ParseUint(claims.Subject, 10, 64)
			if err != nil {
				logger.WithError(err).Debug("failed to parse subject to user id")

				return echo.ErrUnauthorized
			}

			teamRole, _ := c.Get("team_role").(*entities.TeamRole)

			return handler(AuthenticatedContext{
				Context:  c,
				Claims:   *claims,
				Log:      logger,
				UserID:   uint(userId),
				TeamRole: teamRole,
			})
		}
	}
}
