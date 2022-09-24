package middleware

import (
	"github.com/labstack/echo/v4"
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

// Allows only the allowed roles to access the route.
// A team guard must be set before this guard.
func RoleGuardFactory(logger *logrus.Entry, teamService team.Service) func(allowedRoles ...UserRole) func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logrus.WithField("middleware", "role_guard")

	return func(allowedRoles ...UserRole) func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				role, ok := c.Get("team_role").(string)
				if !ok {
					l.Debug("missing team_role, are you missing a team guard?")

					return echo.ErrUnauthorized
				}

				for _, allowedRole := range allowedRoles {
					if role == string(allowedRole) {
						return next(c)
					}
				}

				l.Debug("user role is not allowed")

				return echo.ErrForbidden
			}
		}
	}
}
