package webhooks

import (
	"io"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
)

func (h *Handlers) handleWebhook(c hs.StripeContext) error {
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		c.Log.WithError(err).Debug("failed to read request body for stripe event")

		return echo.ErrBadRequest
	}

	event, err := h.BillingService.ConstructEvent(b, c.Signature)
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
