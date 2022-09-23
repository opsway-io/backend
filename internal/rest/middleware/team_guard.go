package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/sirupsen/logrus"
)

func TeamGuardFactory(logger *logrus.Entry) func() func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logrus.WithField("middleware", "team_guard")

	return func() func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				claims, ok := c.Get("jwt_claims").(*authentication.Claims)
				if !ok {
					l.Debug("missing jwt_claims")

					return echo.ErrUnauthorized
				}
				UserID := claims.Subject
				if UserID == "" {
					l.Debug("missing user_id")
				}

				teamIdParam := c.Param("team_id")
				if teamIdParam == "" {
					l.Debug("missing team_id param")
				}

				l.Debug("Team guard TODO: implement")

				return next(c)
			}
		}
	}
}
