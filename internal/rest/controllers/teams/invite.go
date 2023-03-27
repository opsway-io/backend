package teams

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
)

type PostTeamUsersInvitesRequest struct {
	TeamID uint              `param:"teamId" validate:"required,numeric,gt=0"`
	Email  string            `json:"email" validate:"required,email"`
	Role   entities.TeamRole `json:"role" validate:"required,teamRole"`
}

func (h *Handlers) PostTeamUsersInvites(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostTeamUsersInvitesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostTeamUsersInvitesRequest")

		return echo.ErrBadRequest
	}

	if err := h.TeamService.InviteByEmail(c.Request().Context(), req.TeamID, req.Role, req.Email); err != nil {
		if errors.Is(err, team.ErrAlreadyOnTeam) {
			c.Log.WithError(err).Debug("user is already on team")

			return echo.NewHTTPError(http.StatusConflict, "user is already on team")
		}

		c.Log.WithError(err).Error("failed to invites user to team")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PostTeamInvitesAccept struct {
	Token string `param:"token" validate:"required"`
}

func (h *Handlers) PostTeamInvitesAccept(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostTeamInvitesAccept](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostTeamInvitesAccept")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	user, err := h.UserService.GetUserByID(
		ctx,
		c.UserID,
	)
	if err != nil {
		c.Log.WithError(err).Error("failed to get user")

		return echo.ErrInternalServerError
	}

	if err := h.TeamService.AcceptInviteByToken(
		ctx,
		req.Token,
		user,
	); err != nil {
		c.Log.WithError(err).Error("failed to accept invites")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
