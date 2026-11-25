package middleware

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/platform"
	"github.com/sirupsen/logrus"
)

// ProductGuardFactory restricts route access when a suite product is disabled or restricted for the current user.
// Must be mounted behind AuthGuard.
func ProductGuardFactory(logger *logrus.Entry, platformService platform.Service) func(product entities.ProductSlug) func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logger.WithField("middleware", "product_guard")

	return func(product entities.ProductSlug) func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				userID, ok := extractUserID(c)
				if !ok {
					return echo.ErrUnauthorized
				}

				var teamIDPtr *uint
				if rawTeamID := c.Param("teamId"); rawTeamID != "" {
					if parsed, err := strconv.ParseUint(rawTeamID, 10, 64); err == nil && parsed > 0 {
						t := uint(parsed)
						teamIDPtr = &t
					}
				}

				enabled, err := platformService.IsProductEnabledForUser(c.Request().Context(), product, userID, teamIDPtr)
				if err != nil {
					l.WithError(err).WithField("product", product).Error("failed to check product visibility")
					return echo.ErrInternalServerError
				}
				if !enabled {
					return c.JSON(http.StatusForbidden, map[string]interface{}{
						"code":    "PRODUCT_DISABLED",
						"product": product,
						"message": "This product is not enabled for your account.",
					})
				}

				return next(c)
			}
		}
	}
}
