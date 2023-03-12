package teams

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type PostTeamUsersInviteRequest struct {
	TeamID uint              `param:"teamId" validate:"required,numeric,gt=0"`
	Email  string            `json:"email" validate:"required,email"`
	Role   entities.TeamRole `json:"role" validate:"required,teamRole"`
}

func (h *Handlers) PostTeamUsersInvite(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostTeamUsersInviteRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostTeamUsersInviteRequest")

		return echo.ErrBadRequest
	}

	if err := h.TeamService.InviteByEmail(c.Request().Context(), req.TeamID, req.Email, req.Role); err != nil {
		c.Log.WithError(err).Debug("failed to invite user to team")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
