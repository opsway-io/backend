package middleware

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

// Allows only users in the same team to access the route
func TeamGuardFactory(logger *logrus.Entry, teamService team.Service) func() func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logger.WithField("middleware", "team_guard")

	return func() func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				claims, ok := c.Get("jwt_claims").(*authentication.AccessClaims)
				if !ok {
					l.Debug("missing jwt_claims")

					return echo.ErrForbidden
				}

				userIDStr := claims.Subject
				if userIDStr == "" {
					l.Debug("missing subject in JWT")

					return echo.ErrForbidden
				}

				userID, err := strconv.ParseUint(userIDStr, 10, 64)
				if err != nil {
					l.WithError(err).Debug("failed to parse user ID")

					return echo.ErrForbidden
				}

				teamIDStr := c.Param("teamId")
				if teamIDStr == "" {
					l.Debug("missing team_id param")

					return echo.ErrForbidden
				}

				teamID, err := strconv.ParseUint(teamIDStr, 10, 64)
				if err != nil {
					l.WithError(err).Debug("failed to parse team ID")

					return echo.ErrForbidden
				}

				l = l.WithFields(logrus.Fields{
					"team_id": teamID,
					"user_id": userID,
				})

				userRole, err := teamService.GetUserRole(c.Request().Context(), uint(teamID), uint(userID))
				if err != nil {
					l.WithError(err).Debug("failed to get user role")

					return echo.ErrForbidden
				}

				l.WithField("role", *userRole).Debug("team guard passed")

				c.Set("team_role", userRole)

				return next(c)
			}
		}
	}
}
