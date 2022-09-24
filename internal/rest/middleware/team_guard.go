package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

// Allows only users in the same team to access the route
func TeamGuardFactory(logger *logrus.Entry, teamService team.Service) func() func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logrus.WithField("middleware", "team_guard")

	return func() func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				claims, ok := c.Get("jwt_claims").(*authentication.Claims)
				if !ok {
					l.Debug("missing jwt_claims")

					return echo.ErrForbidden
				}

				UserID := claims.Subject
				if UserID == "" {
					l.Debug("missing subject in JWT")

					return echo.ErrForbidden
				}

				teamIdParam := c.Param("teamId")
				if teamIdParam == "" {
					l.Debug("missing team_id param")

					return echo.ErrForbidden
				}

				l.Debug("Team guard TODO: implement")

				return next(c)
			}
		}
	}
}
