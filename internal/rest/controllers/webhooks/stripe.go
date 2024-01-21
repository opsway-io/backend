package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"strconv"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/team"
	"github.com/stripe/stripe-go/v76"
)

func (h *Handlers) FulfillOrder(context context.Context, lineItems *stripe.LineItemList) {
	fmt.Println(lineItems)

	// TODO: fill me in
}

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
		c.Log.Info("construct")

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
		sessionWithLineItems, err := h.BillingService.GetCheckoutSession(session.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error  getting checkout session")
			return echo.ErrBadRequest
		}

		c.Log.Info(session.ClientReferenceID)
		lineItems := sessionWithLineItems.LineItems
		// Fulfill the purchase...
		customerTeam, err := h.TeamService.GetByStripeID(c.Request().Context(), session.Customer.ID)
		if err != nil {
			if err != team.ErrNotFound {
				return c.NoContent(http.StatusInternalServerError)
			}
			teamID, _ := strconv.ParseUint(session.ClientReferenceID, 10, 32)
			customerTeam, _ := h.TeamService.GetByID(c.Request().Context(), uint(teamID))

			h.TeamService.UpdateBilling(c.Request().Context(), customerTeam.ID, session.Customer.ID, lineItems.Data[0].Price.Product.Name)
			return c.NoContent(http.StatusOK)
		}

		h.TeamService.UpdateBilling(c.Request().Context(), customerTeam.ID, session.Customer.ID, lineItems.Data[0].Price.Product.Name)

		// h.FulfillOrder(lineItems)
		// Payment is successful and the subscription is created.
		// You should provision the subscription and save the customer ID to your database.
	default:
		c.Log.WithField("event", event.Type).Debug("Unhandled event type")
		// unhandled event type
	}

	return c.NoContent(http.StatusOK)
}
