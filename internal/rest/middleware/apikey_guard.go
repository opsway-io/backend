package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/apikey"
	"github.com/sirupsen/logrus"
)

func ApiKeyGuardFactory(logger *logrus.Entry, apiKeyService apikey.Service) func() func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logger.WithField("middleware", "apikey_guard")

	return func() func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				header := c.Request().Header.Get("Authorization")
				if header == "" {
					l.Debug("missing authorization header")
					return echo.ErrUnauthorized
				}

				typ, token, ok := strings.Cut(header, " ")
				if !ok || typ != "Bearer" {
					l.Debug("invalid authorization token type")
					return echo.ErrUnauthorized
				}

				apiKey, err := apiKeyService.GetByPlaintext(c.Request().Context(), token)
				if err != nil {
					l.WithError(err).Debug("failed to verify api key")
					return echo.ErrUnauthorized
				}

				if apiKey == nil {
					l.Debug("invalid api key")
					return echo.ErrUnauthorized
				}

				l.Debug("apikey guard passed")

				c.Set("team_id", apiKey.TeamID)

				return next(c)
			}
		}
	}
}
