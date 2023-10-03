package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// Allows only authenticated users to access the route
func StripeGuardFactory(logger *logrus.Entry) func() func(next echo.HandlerFunc) echo.HandlerFunc {
	l := logger.WithField("middleware", "auth_guard")

	return func() func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				header := c.Request().Header.Get("Stripe-Signature")
				if header == "" {
					l.Debug("missing Stripe-Signature header")

					return echo.ErrUnauthorized
				}

				c.Set("stripe_signature", header)

				return next(c)
			}
		}
	}
}
