package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/sirupsen/logrus"
)

// Allows only authenticated users to access the route
func AuthGuardFactory(logger *logrus.Entry, jwtService authentication.Service) func() func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logger.WithField("middleware", "auth_guard")

	return func() func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				header := c.Request().Header.Get("Authorization")
				if header == "" {
					l.Debug("missing authorization header")

					return echo.ErrUnauthorized
				}

				typ, token, ok := strings.Cut(header, " ")
				if !ok {
					l.Debug("invalid authorization token type")

					return echo.ErrUnauthorized
				}

				if typ != "Bearer" {
					l.Debug("authorization token not Bearer")

					return echo.ErrUnauthorized
				}

				valid, claims, err := jwtService.Verify(c.Request().Context(), token)
				if err != nil {
					l.WithError(err).Debug("failed to verify token")

					return echo.ErrUnauthorized
				}

				if !valid {
					l.Debug("invalid token")

					return echo.ErrUnauthorized
				}

				c.Set("jwt_claims", claims)

				return next(c)
			}
		}
	}
}
