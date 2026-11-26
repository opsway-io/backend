package webhooks

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/notification/email/templates"
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

	ctx := c.Request().Context()

	// Deduplicate retried webhook deliveries idempotently
	if alreadyProcessed, err := h.BillingService.RecordWebhookEvent(ctx, event.ID, string(event.Type)); err == nil && alreadyProcessed {
		c.Log.WithField("event_id", event.ID).Debug("duplicate stripe webhook event ignored")
		return c.NoContent(http.StatusOK)
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			c.Log.WithError(err).Debug("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}
		if session.Subscription == nil || session.Subscription.ID == "" {
			return c.NoContent(http.StatusOK)
		}

		sub, err := h.BillingService.GetSubscription(session.Subscription.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting subscription")
			return c.NoContent(http.StatusInternalServerError)
		}

		teamID, err := strconv.ParseUint(session.ClientReferenceID, 10, 32)
		if err != nil {
			c.Log.WithError(err).Debug("Error parsing team id", session.ClientReferenceID)
			return c.NoContent(http.StatusInternalServerError)
		}

		customerTeam, err := h.TeamService.GetByID(ctx, uint(teamID))
		if err != nil {
			c.Log.WithError(err).Debug("Error getting team by id", teamID)
			return c.NoContent(http.StatusInternalServerError)
		}

		if session.Customer != nil && session.Customer.ID != "" && (customerTeam.StripeCustomerID == nil || *customerTeam.StripeCustomerID == "") {
			customerTeam.StripeCustomerID = &session.Customer.ID
		}

		if session.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid ||
			sub.Status == stripe.SubscriptionStatusActive ||
			sub.Status == stripe.SubscriptionStatusTrialing {
			customerTeam.PaymentPlan = h.BillingService.ResolvePlanFromSubscription(sub)
			if err := h.BillingService.SyncSubscriptionFromStripe(ctx, customerTeam.ID, sub); err != nil {
				c.Log.WithError(err).Error("failed to sync subscription from stripe")
			}
		} else {
			customerTeam.PaymentPlan = entities.PaymentPlanFree
		}

		if err := h.TeamService.UpdateTeam(ctx, customerTeam); err != nil {
			c.Log.WithError(err).Debug("Error updating team")
			return c.NoContent(http.StatusInternalServerError)
		}
		if h.UsageGuard != nil {
			_ = h.UsageGuard.SyncPlanEntitlements(ctx, customerTeam)
		}

	case "customer.subscription.created", "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			c.Log.WithError(err).Debug("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}
		if sub.Customer == nil || sub.Customer.ID == "" {
			return c.NoContent(http.StatusOK)
		}

		team, err := h.TeamService.GetByStripeID(ctx, sub.Customer.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting team by stripe id")
			return c.NoContent(http.StatusInternalServerError)
		}

		if err := h.BillingService.SyncSubscriptionFromStripe(ctx, team.ID, &sub); err != nil {
			c.Log.WithError(err).Error("failed to sync subscription from stripe")
		}
		team.PaymentPlan = h.BillingService.ResolvePlanFromSubscription(&sub)
		if err := h.TeamService.UpdateTeam(ctx, team); err != nil {
			c.Log.WithError(err).Debug("Error updating team")
			return c.NoContent(http.StatusInternalServerError)
		}
		if h.UsageGuard != nil {
			_ = h.UsageGuard.SyncPlanEntitlements(ctx, team)
		}

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			c.Log.WithError(err).Debug("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}
		if sub.Customer == nil || sub.Customer.ID == "" {
			return c.NoContent(http.StatusOK)
		}

		team, err := h.TeamService.GetByStripeID(ctx, sub.Customer.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting team by stripe id")
			return c.NoContent(http.StatusInternalServerError)
		}

		// Revert to FREE unless another active subscription exists for this customer.
		var remainingActive *stripe.Subscription
		if iter := h.BillingService.GetCustomerSubscription(sub.Customer.ID); iter != nil {
			for iter.Next() {
				candidate := iter.Subscription()
				if candidate.ID != sub.ID && (candidate.Status == stripe.SubscriptionStatusActive || candidate.Status == stripe.SubscriptionStatusTrialing) {
					remainingActive = candidate
					break
				}
			}
		}

		if remainingActive != nil {
			_ = h.BillingService.SyncSubscriptionFromStripe(ctx, team.ID, remainingActive)
			team.PaymentPlan = h.BillingService.ResolvePlanFromSubscription(remainingActive)
		} else {
			_ = h.BillingService.DowngradeTeamToFree(ctx, team.ID)
			team.PaymentPlan = entities.PaymentPlanFree
		}

		if err := h.TeamService.UpdateTeam(ctx, team); err != nil {
			c.Log.WithError(err).Debug("Error updating team")
			return c.NoContent(http.StatusInternalServerError)
		}
		if h.UsageGuard != nil {
			_ = h.UsageGuard.SyncPlanEntitlements(ctx, team)
		}

	case "customer.deleted":
		var customer stripe.Customer
		if err := json.Unmarshal(event.Data.Raw, &customer); err != nil {
			c.Log.WithError(err).Debug("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}

		team, err := h.TeamService.GetByStripeID(ctx, customer.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting team by stripe id")
			return c.NoContent(http.StatusInternalServerError)
		}

		_ = h.BillingService.DowngradeTeamToFree(ctx, team.ID)
		team.PaymentPlan = entities.PaymentPlanFree
		team.StripeCustomerID = nil

		if err := h.TeamService.UpdateTeam(ctx, team); err != nil {
			c.Log.WithError(err).Debug("Error updating team")
			return c.NoContent(http.StatusInternalServerError)
		}
		if h.UsageGuard != nil {
			_ = h.UsageGuard.SyncPlanEntitlements(ctx, team)
		}

	case "invoice.paid":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			return echo.ErrBadRequest
		}
		if invoice.Customer != nil && invoice.Customer.ID != "" {
			if team, err := h.TeamService.GetByStripeID(ctx, invoice.Customer.ID); err == nil && team != nil {
				var nextReset time.Time
				if invoice.Subscription != nil && invoice.Subscription.ID != "" {
					if sub, err := h.BillingService.GetSubscription(invoice.Subscription.ID); err == nil && sub != nil {
						_ = h.BillingService.SyncSubscriptionFromStripe(ctx, team.ID, sub)
						if sub.CurrentPeriodEnd > 0 {
							nextReset = time.Unix(sub.CurrentPeriodEnd, 0).UTC()
						}
					}
				}
				_ = h.BillingService.ResetPeriodUsage(ctx, team.ID, nextReset)
				if h.UsageGuard != nil {
					h.UsageGuard.ReconcileTeamStripePlan(ctx, team)
				}
			}
		}

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			c.Log.WithError(err).Debug("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}
		if invoice.Customer == nil || invoice.Customer.ID == "" {
			return c.NoContent(http.StatusOK)
		}

		team, err := h.TeamService.GetByStripeID(ctx, invoice.Customer.ID)
		if err != nil {
			c.Log.WithError(err).Debug("Error getting team by stripe id")
			return c.NoContent(http.StatusInternalServerError)
		}

		if h.UsageGuard != nil {
			h.UsageGuard.ReconcileTeamStripePlan(ctx, team)
		}

		// Notify team owner (fallback to any team member if no owner found)
		ownerRole := entities.TeamRoleOwner
		offset, limit := 0, 5
		users, err := h.TeamService.GetUsersByID(ctx, team.ID, &offset, &limit, nil, &ownerRole)
		if err != nil || users == nil || len(*users) == 0 {
			users, err = h.TeamService.GetUsersByID(ctx, team.ID, &offset, &limit, nil, nil)
		}
		if err == nil && users != nil && len(*users) > 0 {
			billingURL := h.BillingService.Domain() + "/settings/billing"
			for _, owner := range *users {
				_ = h.EmailSender.Send(
					ctx,
					"",
					owner.Email,
					&templates.BillingAlertTemplate{
						TeamName:   team.Name,
						Message:    "We failed to process the payment for your recent invoice.",
						BillingURL: billingURL,
					},
				)
			}
		}

	default:
		c.Log.WithField("event", event.Type).Debug("Unhandled event type")
	}

	return c.NoContent(http.StatusOK)
}
