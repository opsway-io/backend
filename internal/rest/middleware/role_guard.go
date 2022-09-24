package middleware

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type UserRole string

const (
	// Admin is a user with full access to the organization.
	UserRoleAdmin UserRole = "admin"

	// Members can view and act on events, as well as view most other data within the organization.
	UserRoleMember UserRole = "member"
)

// Allows only the allowed roles to access the route
func RoleGuardFactory(logger *logrus.Entry, teamService team.Service) func(allowedRoles ...UserRole) func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logrus.WithField("middleware", "role_guard")

	return func(allowedRoles ...UserRole) func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				claims, ok := c.Get("jwt_claims").(*authentication.Claims)
				if !ok {
					l.Debug("missing jwt_claims")

					return echo.ErrUnauthorized
				}

				UserIDStr := claims.Subject
				if UserIDStr == "" {
					l.Debug("missing subject in JWT")

					return echo.ErrForbidden
				}

				UserID, err := strconv.ParseUint(UserIDStr, 10, 64)
				if err != nil {
					l.WithError(err).Debug("invalid user id")

					return echo.ErrForbidden
				}

				teamIDStr := c.Param("teamId")
				if teamIDStr == "" {
					l.Debug("missing team_id param")

					return echo.ErrForbidden
				}

				teamID, err := strconv.ParseUint(teamIDStr, 10, 64)
				if err != nil {
					l.Debug("invalid team_id param")

					return echo.ErrForbidden
				}

				role, err := teamService.GetUserRole(c.Request().Context(), uint(teamID), uint(UserID))
				if err != nil {
					l.WithError(err).Debug("failed to get user role")

					return echo.ErrForbidden
				}

				for _, allowedRole := range allowedRoles {
					if *role == entities.Role(allowedRole) {
						return next(c)
					}
				}

				l.Debug("user role is not allowed")

				return echo.ErrForbidden
			}
		}
	}
}
