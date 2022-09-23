package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/sirupsen/logrus"
)

func RoleGuardFactory(logger *logrus.Entry) func(allowedRoles ...string) func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logrus.WithField("middleware", "role_guard")

	return func(allowedRoles ...string) func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				claims, ok := c.Get("jwt_claims").(*authentication.Claims)
				if !ok {
					l.Debug("missing jwt_claims")

					return echo.ErrUnauthorized
				}

				l.Debug(claims) // TODO: implement

				return next(c)
			}
		}
	}
}
