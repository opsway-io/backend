package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/sirupsen/logrus"
)

// Allows only authenticated users to access the route
func AuthGuardFactory(logger *logrus.Entry, jwtService authentication.Service) func() func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logger.WithField("middleware", "auth_guard")

	return func() func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				token, ok := getAccessToken(c)
				if !ok {
					l.Debug("missing or invalid access token")

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

				l.Debug("auth guard passed")

				c.Set("jwt_claims", claims)

				return next(c)
			}
		}
	}
}

// Reads the access token from the Authorization header, falling back to the access token cookie
func getAccessToken(c echo.Context) (string, bool) {
	header := c.Request().Header.Get("Authorization")
	if header == "" {
		cookie, err := c.Cookie(helpers.AccessTokenCookieName)
		if err != nil || cookie.Value == "" {
			return "", false
		}

		return cookie.Value, true
	}

	typ, token, ok := strings.Cut(header, " ")
	if !ok || typ != "Bearer" {
		return "", false
	}

	return token, true
}
