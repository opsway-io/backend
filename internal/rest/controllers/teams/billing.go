package teams

import (
	"net/http"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/billing"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/stripe/stripe-go/v81"
)

func (h *Handlers) PostConfig(c hs.AuthenticatedContext) error {
	return c.JSON(http.StatusOK, h.BillingService.PostConfig())
}

type PostCreateCheckoutSession struct {
	TeamID    uint   `param:"teamId" validate:"required,numeric,gt=0"`
	Plan      string `json:"plan" validate:"required,max=255"`
	ReturnURL string `json:"returnUrl"`
}

func (h *Handlers) PostCreateCheckoutSession(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostCreateCheckoutSession](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostCreateCheckoutSession")
		return echo.ErrBadRequest
	}

	team, err := h.TeamService.GetByID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Debug("Team not found")
		return echo.ErrInternalServerError
	}

	targetTier := entities.NormalizeTier(req.Plan)
	currentTier := entities.NormalizeTier(string(team.PaymentPlan))

	if currentTier == targetTier {
		// If currently scheduled to cancel and user re-selects current paid plan, resume it
		if targetTier != entities.EntitlementTierFree && team.StripeCustomerID != nil && *team.StripeCustomerID != "" {
			if _, err := h.BillingService.ResumeSubscription(team); err != nil {
				c.Log.WithError(err).Debug("resume subscription")
			}
		}
		return c.JSON(http.StatusOK, "")
	}

	priceID := ""
	switch targetTier {
	case entities.EntitlementTierPro:
		priceID = h.BillingService.PostConfig().ProPrice
	case entities.EntitlementTierEnterprise:
		priceID = h.BillingService.PostConfig().EnterprisePrice
	}

	if team.StripeCustomerID == nil || *team.StripeCustomerID == "" {
		if targetTier == entities.EntitlementTierFree {
			return c.JSON(http.StatusOK, "")
		}
		s, err := h.BillingService.CreateCheckoutSessionWithReturn(team, priceID, req.ReturnURL, nil)
		if err != nil {
			c.Log.WithError(err).Debug("create stripe checkout session")
			return echo.ErrInternalServerError
		}
		return c.JSON(http.StatusOK, s.URL)
	}

	if targetTier == entities.EntitlementTierFree {
		_, err := h.BillingService.CancelSubscription(team)
		if err != nil {
			c.Log.WithError(err).Debug("cancel subscription")
			return echo.ErrInternalServerError
		}
		return c.JSON(http.StatusOK, "")
	}

	if currentTier == entities.EntitlementTierFree {
		s, err := h.BillingService.CreateCheckoutSessionWithReturn(team, priceID, req.ReturnURL, nil)
		if err != nil {
			c.Log.WithError(err).Debug("create stripe checkout session")
			return echo.ErrInternalServerError
		}
		return c.JSON(http.StatusOK, s.URL)
	}

	_, err = h.BillingService.UpdateSubscription(team, priceID)
	if err != nil {
		c.Log.WithError(err).Debug("update subscription")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, "")
}

type GetCheckoutSession struct {
	SessionID string `param:"SessionID" validate:"required"`
}

func (h *Handlers) GetCheckoutSession(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetCheckoutSession](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetCheckoutSession")
		return echo.ErrBadRequest
	}
	s, err := h.BillingService.GetCheckoutSession(req.SessionID)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get session")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, s)
}

type GetCustomerPortalRequest struct {
	TeamID    uint   `param:"teamId" validate:"required,numeric,gt=0"`
	ReturnURL string `json:"returnUrl"`
}

type GetCustomerPortalResponse struct {
	URL string `json:"url"`
}

func (h *Handlers) PostCustomerPortal(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetCustomerPortalRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetCustomerPortalRequest")
		return echo.ErrBadRequest
	}

	team, err := h.TeamService.GetByID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Debug("Team not found")
		return echo.ErrInternalServerError
	}

	if team.StripeCustomerID == nil || *team.StripeCustomerID == "" {
		return c.JSON(http.StatusOK, GetCustomerPortalResponse{URL: ""})
	}

	ps, err := h.BillingService.CreateCustomerPortalWithReturn(team, req.ReturnURL)
	if err != nil {
		c.Log.WithError(err).Debug("failed to create customer portal")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, GetCustomerPortalResponse{URL: ps.URL})
}

type GetCustomerSession struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetCustomerSessionResponse struct {
	SessionID string `json:"sessionId"`
}

func (h *Handlers) GetCustomerSession(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetCustomerSession](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetCustomerSession")
		return echo.ErrBadRequest
	}

	team, err := h.TeamService.GetByID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Debug("Team not found")
		return echo.ErrInternalServerError
	}

	if team.StripeCustomerID == nil || *team.StripeCustomerID == "" {
		return c.JSON(http.StatusOK, GetCustomerSessionResponse{SessionID: ""})
	}

	s, err := h.BillingService.GetCustomerSession(team)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get session")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, GetCustomerSessionResponse{SessionID: s.ClientSecret})
}

type GetProductsResponse struct {
	Products []Product `json:"products"`
}

type Product struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Price     int64    `json:"price"`
	Currency  string   `json:"currency"`
	LookupKey string   `json:"lookupKey"`
	Features  []string `json:"marketing_features"`
}

func (h *Handlers) GetProducts(c hs.AuthenticatedContext) error {
	catalog := h.BillingService.GetCatalog()

	// If Stripe is configured, enrich with live Stripe prices; otherwise serve canonical EUR catalog.
	if h.BillingService.IsConfigured() {
		p := h.BillingService.GetProducts()
		prices, err := h.BillingService.GetPrices([]string{"free", "pro", "team", "enterprise"})
		if err == nil && len(prices) > 0 && p != nil {
			priceMap := make(map[string]*stripe.Price, len(prices))
			for _, pr := range prices {
				priceMap[pr.ID] = pr
			}
			products := make([]Product, 0)
			for _, stripeProduct := range p.ProductList().Data {
				if stripeProduct.DefaultPrice == nil {
					continue
				}
				pr, ok := priceMap[stripeProduct.DefaultPrice.ID]
				if !ok {
					continue
				}
				lookup := strings.ToLower(pr.LookupKey)
				if lookup == "team" {
					lookup = "pro"
				}
				prod := Product{
					ID:        stripeProduct.ID,
					Name:      stripeProduct.Name,
					Price:     pr.UnitAmount / 100,
					Currency:  string(pr.Currency),
					LookupKey: lookup,
					Features:  make([]string, 0, len(stripeProduct.MarketingFeatures)),
				}
				for _, feature := range stripeProduct.MarketingFeatures {
					prod.Features = append(prod.Features, feature.Name)
				}
				products = append(products, prod)
			}
			if len(products) > 0 {
				sort.Slice(products, func(i, j int) bool {
					return products[i].Price < products[j].Price
				})
				return c.JSON(http.StatusOK, GetProductsResponse{Products: products})
			}
		}
	}

	// Canonical EUR catalog fallback
	products := make([]Product, 0, len(catalog.Plans))
	for _, plan := range catalog.Plans {
		products = append(products, Product{
			ID:        string(plan.ID),
			Name:      plan.Name,
			Price:     plan.Price,
			Currency:  plan.Currency,
			LookupKey: plan.LookupKey,
			Features:  plan.Features,
		})
	}
	return c.JSON(http.StatusOK, GetProductsResponse{Products: products})
}

type BillingSummaryRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type BillingSummaryResponse struct {
	TeamID           uint                       `json:"teamId"`
	PaymentPlan      entities.EntitlementTier   `json:"paymentPlan"`
	CanManageBilling bool                       `json:"canManageBilling"`
	Subscription     *entities.TeamSubscription `json:"subscription,omitempty"`
	Entitlements     []entities.TeamEntitlement `json:"entitlements"`
	Catalog          billing.BillingCatalog     `json:"catalog"`
}

func (h *Handlers) GetBillingSummary(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[BillingSummaryRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	ctx := c.Request().Context()
	team, err := h.TeamService.GetByID(ctx, req.TeamID)
	if err != nil {
		return echo.ErrInternalServerError
	}

	role, _ := h.TeamService.GetUserRole(ctx, req.TeamID, c.UserID)
	canManage := role != nil && *role == entities.TeamRoleOwner

	subRec, _ := h.BillingService.GetSubscriptionRecord(ctx, req.TeamID)
	ents, err := h.BillingService.GetEntitlements(ctx, req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get team entitlements")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, BillingSummaryResponse{
		TeamID:           team.ID,
		PaymentPlan:      entities.NormalizeTier(string(team.PaymentPlan)),
		CanManageBilling: canManage,
		Subscription:     subRec,
		Entitlements:     ents,
		Catalog:          h.BillingService.GetCatalog(),
	})
}

type PostProductAddonRequest struct {
	TeamID    uint                     `param:"teamId" validate:"required,numeric,gt=0"`
	Product   entities.ProductSlug     `json:"product" validate:"required"`
	Tier      entities.EntitlementTier `json:"tier" validate:"required"`
	ReturnURL string                   `json:"returnUrl"`
}

type PostProductAddonResponse struct {
	CheckoutURL string `json:"checkoutUrl,omitempty"`
}

func (h *Handlers) PostProductAddon(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostProductAddonRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	if !entities.IsValidProduct(req.Product) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid product")
	}

	ctx := c.Request().Context()
	team, err := h.TeamService.GetByID(ctx, req.TeamID)
	if err != nil {
		return echo.ErrInternalServerError
	}

	checkoutURL, err := h.BillingService.UpsertProductAddon(ctx, team, req.Product, req.Tier, req.ReturnURL)
	if err != nil {
		c.Log.WithError(err).Warn("failed to upsert product addon")
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, PostProductAddonResponse{CheckoutURL: checkoutURL})
}

func (h *Handlers) PostResumeSubscription(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[BillingSummaryRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	ctx := c.Request().Context()
	team, err := h.TeamService.GetByID(ctx, req.TeamID)
	if err != nil {
		return echo.ErrInternalServerError
	}
	if _, err := h.BillingService.ResumeSubscription(team); err != nil {
		c.Log.WithError(err).Warn("failed to resume subscription")
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
