package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/sirupsen/logrus"
)

// Allows only the current user to access the route
func CurrentUserGuardFactory(logger *logrus.Entry) func() func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logger.WithField("middleware", "current_user_guard")

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

				userIdParam := c.Param("userId")
				if userIdParam == "" {
					l.Debug("missing userId param")

					return echo.ErrForbidden
				}

				if userIdParam != claims.Subject {
					l.Debug("User id in request does not match authenticated user id")

					return echo.ErrForbidden
				}

				l.Debug("current user guard passed")

				return next(c)
			}
		}
	}
}
