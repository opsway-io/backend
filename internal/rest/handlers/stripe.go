package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type StripeHandlerFunc func(c StripeContext) error

type StripeContext struct {
	echo.Context
	Log       *logrus.Entry
	Signature string
}

func StripeHandlerFactory(logger *logrus.Entry) func(handler StripeHandlerFunc) func(c echo.Context) error {
	return func(handler StripeHandlerFunc) func(c echo.Context) error {
		return func(c echo.Context) error {
			signature, ok := c.Get("stripe_signature").(string)

			if !ok {
				logger.Info("Stripe-Signature")

				return echo.ErrUnauthorized
			}

			return handler(StripeContext{
				Context:   c,
				Log:       logger,
				Signature: signature,
			})
		}
	}
}
