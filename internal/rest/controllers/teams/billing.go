package teams

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

func (h *Handlers) PostConfig(c hs.AuthenticatedContext) error {
	return c.JSON(http.StatusOK, h.BillingService.PostConfig())
}

type PostCreateCheckoutSession struct {
	TeamID         uint   `param:"teamId" validate:"required,numeric,gt=0"`
	PriceLookupKey string `json:"priceLookupKey" validate:"required,max=255"`
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

	if team.PaymentPlan == entities.PaymentPlan(req.PriceLookupKey) {
		return c.JSON(http.StatusOK, "")
	}

	if team.StripeCustomerID == nil {
		if req.PriceLookupKey == "FREE" {
			return c.JSON(http.StatusOK, "")
		}
		s, err := h.BillingService.CreateCheckoutSession(team, req.PriceLookupKey)
		if err != nil {
			c.Log.WithError(err).Debug("create stripe checkout session")

			return echo.ErrInternalServerError
		}
		return c.JSON(http.StatusOK, s.URL)
	}

	if req.PriceLookupKey == "FREE" {
		_, err := h.BillingService.CancelSubscribtion(team)
		if err != nil {
			c.Log.WithError(err).Debug("cancel subscription")

			return echo.ErrInternalServerError
		}
		return c.JSON(http.StatusOK, "")
	}

	if team.PaymentPlan == "FREE" {
		s, err := h.BillingService.CreateCheckoutSession(team, req.PriceLookupKey)
		if err != nil {
			c.Log.WithError(err).Debug("create stripe checkout session")

			return echo.ErrInternalServerError
		}
		return c.JSON(http.StatusOK, s.URL)
	}

	_, err = h.BillingService.UpdateSubscribtion(team, req.PriceLookupKey)
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
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
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

	ps, err := h.BillingService.CreateCustomerPortal(team)
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

	s, err := h.BillingService.GetCustomerSession(team)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get session")

		return echo.ErrInternalServerError
	}

	fmt.Println("TEEEEEEEEEEEEEEEEEEEEEe SESSION")

	return c.JSON(http.StatusOK, GetCustomerSessionResponse{SessionID: s.ClientSecret})
}
