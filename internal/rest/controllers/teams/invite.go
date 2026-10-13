package teams

import (
	"errors"
	"net/http"
	"net/url"
	"time"

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
	Token string `json:"token" validate:"required"`
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
		if errors.Is(err, team.ErrInvalidInvitationToken) {
			return echo.NewHTTPError(http.StatusBadRequest, "invitation is no longer valid")
		}

		c.Log.WithError(err).Error("failed to accept invites")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type GetTeamUsersInvitesRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamUsersInvitesResponse struct {
	Invites []GetTeamUsersInvitesResponseInvite `json:"invites"`
}

type GetTeamUsersInvitesResponseInvite struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *Handlers) GetTeamUsersInvites(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetTeamUsersInvitesRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	invites, err := h.TeamService.GetInvites(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get invites")

		return echo.ErrInternalServerError
	}

	resp := GetTeamUsersInvitesResponse{
		Invites: make([]GetTeamUsersInvitesResponseInvite, len(*invites)),
	}
	for i, inv := range *invites {
		resp.Invites[i] = GetTeamUsersInvitesResponseInvite{
			ID:        inv.ID,
			Email:     inv.Email,
			Role:      string(inv.Role),
			CreatedAt: inv.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, resp)
}

type DeleteTeamUsersInviteRequest struct {
	TeamID uint   `param:"teamId" validate:"required,numeric,gt=0"`
	Email  string `param:"email" validate:"required"`
}

func (h *Handlers) DeleteTeamUsersInvite(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteTeamUsersInviteRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	email, err := url.PathUnescape(req.Email)
	if err != nil {
		return echo.ErrBadRequest
	}

	if err := h.TeamService.DeleteInvite(c.Request().Context(), req.TeamID, email); err != nil {
		if errors.Is(err, team.ErrInviteNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to delete invite")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
