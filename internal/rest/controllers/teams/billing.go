package teams

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type PostCreateCheckoutSession struct {
	PriceID string `param:"priceId" validate:"required"`
}

func (h *Handlers) PostCreateCheckoutSession(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostCreateCheckoutSession](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetTeamUsersRequest")

		return echo.ErrBadRequest
	}

	s, err := h.BillingService.CreateSession(req.PriceID)
	if err != nil {
		c.Log.WithError(err).Debug("create stripe checkout session")

		return echo.ErrInternalServerError
	}

	return c.Redirect(http.StatusSeeOther, s.SuccessURL)
}
