package teams

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type PostTeamAvailableRequest struct {
	Name string `json:"name"`
}

type PostTeamAvailableResponse struct {
	Available bool `json:"available"`
}

func (h *Handlers) PostTeamAvailable(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostTeamAvailableRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostTeamAvailableRequest")

		return echo.ErrBadRequest
	}

	available, err := h.TeamService.IsNameAvailable(c.Request().Context(), req.Name)
	if err != nil {
		c.Log.WithError(err).Debug("failed to check if team name is available")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, PostTeamAvailableResponse{
		Available: available,
	})
}
