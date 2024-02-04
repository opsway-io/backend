package webhooks

import (
	"encoding/json"
	"io"
	"net/http"

	"strconv"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/stripe/stripe-go/v76"
)

func (h *Handlers) handleWebhook(c hs.StripeContext) error {
	c.Log.Info("stripe webhook received")
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		c.Log.WithError(err).Debug("failed to read request body for stripe event")

		return echo.ErrBadRequest
	}

	event, err := h.BillingService.ConstructEvent(b, c.Signature)
	if err != nil {
		c.Log.WithError(err).Info("failed to construct stripe event")

		return echo.ErrBadRequest
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			c.Log.WithError(err).Debug("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}

		params := &stripe.CheckoutSessionParams{}
		params.AddExpand("line_items")

		// Retrieve the session. If you require line items in the response, you may include them by expanding line_items.
		// sessionWithLineItems, err := h.BillingService.GetCheckoutSession(session.ID)
		// if err != nil {
		// 	c.Log.WithError(err).Debug("Error  getting checkout session")
		// 	return echo.ErrBadRequest
		// }

		c.Log.Error(session.ClientReferenceID)
		c.Log.Error(session)
		c.Log.Error(session.ID)
		c.Log.Error(event.Data.Object["customer"].(string))
		lineItems := h.BillingService.GetLineItems(session.ID).List()
		c.Log.Error(lineItems)

		teamID, err := strconv.ParseUint(session.ClientReferenceID, 10, 32)
		if err != nil {
			c.Log.WithError(err).Debug("Error parsing team id", session.ClientReferenceID)
			return c.NoContent(http.StatusInternalServerError)
		}

		c.Log.Error(c.Request().Context())
		c.Log.Error(uint(teamID))
		customerTeam, err := h.TeamService.GetByID(c.Request().Context(), uint(teamID))
		if err != nil {
			c.Log.WithError(err).Debug("Error getting team by id", teamID)
			return c.NoContent(http.StatusInternalServerError)
		}

		// TODO if not same plan remove old plan

		// if customerTeam.PaymentPlan == "TEST" { //lineItems.Price.Product.Name {
		// 	return c.NoContent(http.StatusOK)
		// }

		customerID := event.Data.Object["customer"].(string)
		if customerTeam.StripeCustomerID == nil {
			customerTeam.StripeCustomerID = &customerID
		}

		err = h.TeamService.UpdateTeam(c.Request().Context(), customerTeam)
		if err != nil {
			c.Log.WithError(err).Debug("Error updating team")
			return c.NoContent(http.StatusInternalServerError)
		}
	case "customer.subscription.updated":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			c.Log.WithError(err).Debug("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}
		c.Log.Info(*subscription.Items.Data[0])
		// c.Log.Info(subscription.Plan)
		c.Log.Info("customer.subscription.updated")

	default:
		c.Log.WithField("event", event.Type).Debug("Unhandled event type")
		// unhandled event type
	}

	return c.NoContent(http.StatusOK)
}
