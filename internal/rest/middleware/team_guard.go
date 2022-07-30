package middleware

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/jwt"
	"github.com/sirupsen/logrus"
)

func TeamGuard(logger *logrus.Entry) echo.MiddlewareFunc {
	l := logrus.WithField("middleware", "team_guard")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := c.Get("jwt_claims").(*jwt.Claims)
			if !ok {
				l.Debug("missing jwt_claims")

				return echo.ErrUnauthorized
			}

			if claims.TeamID == 0 {
				l.Debug("missing team_id id claims")

				return echo.ErrUnauthorized
			}

			teamIdParam := c.Param("team_id")
			if teamIdParam == "" {
				l.Debug("missing team_id param")
			}

			if teamIdParam != strconv.Itoa(claims.TeamID) {
				l.Debug("team_id param does not match claims")

				return echo.ErrUnauthorized
			}

			return next(c)
		}
	}
}
