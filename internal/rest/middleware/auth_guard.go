package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/sirupsen/logrus"
)

// Allows only authenticated users to access the route
func AuthGuardFactory(logger *logrus.Entry, cookieService helpers.CookieService, jwtService authentication.Service) func() func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logger.WithField("middleware", "auth_guard")

	return func() func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				cookie, err := cookieService.GetAccessToken(c)
				if err != nil {
					l.WithError(err).Debug("failed to get access token cookie")

					return echo.ErrUnauthorized
				}

				token := cookie.Value

				valid, claims, err := jwtService.Verify(c.Request().Context(), token)
				if err != nil {
					l.WithError(err).Debug("failed to verify token")

					return echo.ErrUnauthorized
				}

				if !valid {
					l.Debug("invalid token")

					return echo.ErrUnauthorized
				}

				l.Debug("auth guard passed")

				c.Set("jwt_claims", claims)

				return next(c)
			}
		}
	}
}
