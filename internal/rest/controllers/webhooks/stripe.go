package webhooks

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/stripe/stripe-go/v81"
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

		subscription, err := h.BillingService.GetSubscribtion(session.Subscription.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting subscription")
			return c.NoContent(http.StatusInternalServerError)
		}

		product, err := h.BillingService.GetProduct(subscription.Items.Data[0].Price.Product.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting product")
			return c.NoContent(http.StatusInternalServerError)
		}

		teamID, err := strconv.ParseUint(session.ClientReferenceID, 10, 32)
		if err != nil {
			c.Log.WithError(err).Debug("Error parsing team id", session.ClientReferenceID)
			return c.NoContent(http.StatusInternalServerError)
		}

		customerTeam, err := h.TeamService.GetByID(c.Request().Context(), uint(teamID))
		if err != nil {
			c.Log.WithError(err).Debug("Error getting team by id", teamID)
			return c.NoContent(http.StatusInternalServerError)
		}

		customerTeam.PaymentPlan = entities.PaymentPlan(strings.ToUpper(product.Name))
		if customerTeam.StripeCustomerID == nil || *customerTeam.StripeCustomerID == "" {
			customerTeam.StripeCustomerID = &session.Customer.ID
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

		team, err := h.TeamService.GetByStripeID(c.Request().Context(), subscription.Customer.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting team by stripe id")
			return c.NoContent(http.StatusInternalServerError)
		}

		product, err := h.BillingService.GetProduct(subscription.Items.Data[0].Price.Product.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting product")
			return c.NoContent(http.StatusInternalServerError)
		}

		team.PaymentPlan = entities.PaymentPlan(strings.ToUpper(product.Name))

		if subscription.Status == "canceled" {
			team.PaymentPlan = "FREE"
		}

		err = h.TeamService.UpdateTeam(c.Request().Context(), team)
		if err != nil {
			c.Log.WithError(err).Debug("Error updating team")
			return c.NoContent(http.StatusInternalServerError)
		}

	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			c.Log.WithError(err).Debug("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}

		team, err := h.TeamService.GetByStripeID(c.Request().Context(), subscription.Customer.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting team by stripe id")
			return c.NoContent(http.StatusInternalServerError)
		}
		team.PaymentPlan = entities.PaymentPlan(subscription.Items.Data[0].Price.LookupKey)

		if subscription.Status == "canceled" {
			team.PaymentPlan = "FREE"
		}

		err = h.TeamService.UpdateTeam(c.Request().Context(), team)
		if err != nil {
			c.Log.WithError(err).Debug("Error updating team")
			return c.NoContent(http.StatusInternalServerError)
		}

	case "customer.deleted":
		var customer stripe.Customer
		err := json.Unmarshal(event.Data.Raw, &customer)
		if err != nil {
			c.Log.WithError(err).Debug("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}

		team, err := h.TeamService.GetByStripeID(c.Request().Context(), customer.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting team by stripe id")
			return c.NoContent(http.StatusInternalServerError)
		}

		team.PaymentPlan = "FREE"
		team.StripeCustomerID = nil

		err = h.TeamService.UpdateTeam(c.Request().Context(), team)
		if err != nil {
			c.Log.WithError(err).Debug("Error updating team")
			return c.NoContent(http.StatusInternalServerError)
		}

	default:
		c.Log.WithField("event", event.Type).Debug("Unhandled event type")
		// unhandled event type
	}

	return c.NoContent(http.StatusOK)
}
