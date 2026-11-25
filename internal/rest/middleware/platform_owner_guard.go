package middleware

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/platform"
	"github.com/sirupsen/logrus"
)

func extractUserID(c echo.Context) (uint, bool) {
	claims, ok := c.Get("jwt_claims").(*authentication.AccessClaims)
	if !ok || claims.Subject == "" {
		return 0, false
	}
	parsed, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || parsed == 0 {
		return 0, false
	}
	return uint(parsed), true
}

// PlatformOwnerGuardFactory restricts access to users with is_platform_owner = true.
// Must be mounted behind AuthGuard.
func PlatformOwnerGuardFactory(logger *logrus.Entry, platformService platform.Service) func() func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logger.WithField("middleware", "platform_owner_guard")

	return func() func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				userID, ok := extractUserID(c)
				if !ok {
					l.Debug("missing or invalid jwt_claims")
					return echo.ErrForbidden
				}

				isOwner, err := platformService.IsPlatformOwner(c.Request().Context(), userID)
				if err != nil {
					l.WithError(err).Error("failed to check platform owner status")
					return echo.ErrInternalServerError
				}
				if !isOwner {
					l.WithField("user_id", userID).Debug("user is not a platform owner")
					return echo.ErrForbidden
				}

				return next(c)
			}
		}
	}
}
