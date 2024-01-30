package teams

import (
	"net/http"

	"github.com/labstack/echo/v4"
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
	c.Log.Error(req.PriceLookupKey)
	c.Log.Info(req.TeamID)

	team, err := h.TeamService.GetByID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Debug("Team not found")

		return echo.ErrInternalServerError
	}

	s, err := h.BillingService.CreateCheckoutSession(team, req.PriceLookupKey)
	if err != nil {
		c.Log.WithError(err).Debug("create stripe checkout session")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, s.URL)
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

type PostCustomerPortal struct {
	SessionID string `param:"SessionID" validate:"required"`
}

func (h *Handlers) PostCustomerPortal(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostCustomerPortal](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostCustomerPortal")

		return echo.ErrBadRequest
	}

	ps, err := h.BillingService.CreateCustomerPortal(req.SessionID)
	if err != nil {
		c.Log.WithError(err).Debug("failed to create customer portal")

		return echo.ErrInternalServerError
	}
	return c.Redirect(http.StatusSeeOther, ps.URL)
}
