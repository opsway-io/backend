package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type UserRole string

const (
	// Owner is the owner of the team and has full access to the team.
	UserRoleOwner UserRole = "OWNER"

	// Admin is a user with full administrative access to the team.
	UserRoleAdmin UserRole = "ADMIN"

	// Members can view and act on events, as well as view most other data within the organization.
	UserRoleMember UserRole = "MEMBER"
)

// Allows only the allowed roles to access the route.
// A team guard must be set before this guard.
func RoleGuardFactory(logger *logrus.Entry, teamService team.Service) func(allowedRoles ...UserRole) func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logger.WithField("middleware", "role_guard")

	return func(allowedRoles ...UserRole) func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				teamRole, ok := c.Get("team_role").(*entities.TeamRole)
				if !ok {
					l.Debug("missing team_role, are you missing a team guard?")

					return echo.ErrUnauthorized
				}

				for _, allowedRole := range allowedRoles {
					if string(*teamRole) == string(allowedRole) {
						l.Debug("role guard passed")

						c.Set("team_role", teamRole)

						return next(c)
					}
				}

				l.Debug("user does not have required role")

				return echo.ErrForbidden
			}
		}
	}
}
