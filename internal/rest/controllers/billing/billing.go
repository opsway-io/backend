package billing

import (
	"io/ioutil"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	stripe "github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/webhook"
)

// Set your secret key. Remember to switch to your live secret key in production.
// See your keys here: https://dashboard.stripe.com/apikeys

func (h *Handlers) handleWebhook(c hs.StripeContext) error {
	stripe.Key = "test"

	b, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostTeamAvailableRequest")

		return echo.ErrBadRequest
	}

	endpointSecret := "test"
	event, err := webhook.ConstructEvent(b, c.Signature, endpointSecret)
	if err != nil {
		c.Log.WithError(err).Debug("failed to construct stripe event")

		return echo.ErrBadRequest
	}

	switch event.Type {
	case "checkout.session.completed":
		// Payment is successful and the subscription is created.
		// You should provision the subscription and save the customer ID to your database.
	case "invoice.paid":
		// Continue to provision the subscription as payments continue to be made.
		// Store the status in your database and check when a user accesses your service.
		// This approach helps you avoid hitting rate limits.
	case "invoice.payment_failed":
		// The payment failed or the customer does not have a valid payment method.
		// The subscription becomes past_due. Notify your customer and send them to the
		// customer portal to update their payment information.
	default:
		// unhandled event type
	}

	return nil
}