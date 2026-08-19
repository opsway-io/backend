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

	ctx := c.Request().Context()
	teamEntity, err := h.TeamService.GetByID(ctx, req.TeamID)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get team")
		return echo.ErrInternalServerError
	}

	limit := teamEntity.GetTeamMemberLimit()
	if limit != -1 {
		count, err := h.TeamService.GetTeamUserCount(ctx, req.TeamID)
		if err != nil {
			c.Log.WithError(err).Debug("failed to get team user count")
			return echo.ErrInternalServerError
		}
		if count >= int64(limit) {
			return echo.NewHTTPError(http.StatusPaymentRequired, "Payment Required")
		}
	}

	if err := h.TeamService.InviteByEmail(ctx, req.TeamID, req.Role, req.Email); err != nil {
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
		c.Log.WithError(err).Error("failed to accept invites")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type GetTeamUsersInvitesRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamUsersInvitesResponse struct {
	Invites []GetTeamUsersInviteResponse `json:"invites"`
}

type GetTeamUsersInviteResponse struct {
	Email     string            `json:"email"`
	Role      entities.TeamRole `json:"role"`
	CreatedAt string            `json:"createdAt"`
}

func (h *Handlers) GetTeamUsersInvites(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetTeamUsersInvitesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetTeamUsersInvitesRequest")
		return echo.ErrBadRequest
	}

	invites, err := h.TeamService.GetTeamInvitations(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get team invitations")
		return echo.ErrInternalServerError
	}

	res := GetTeamUsersInvitesResponse{
		Invites: make([]GetTeamUsersInviteResponse, len(*invites)),
	}

	for i, invite := range *invites {
		res.Invites[i] = GetTeamUsersInviteResponse{
			Email:     invite.Email,
			Role:      invite.Role,
			CreatedAt: invite.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return c.JSON(http.StatusOK, res)
}

type DeleteTeamUsersInvitesRequest struct {
	TeamID uint   `param:"teamId" validate:"required,numeric,gt=0"`
	Email  string `param:"email" validate:"required,email"`
}

func (h *Handlers) DeleteTeamUsersInvites(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteTeamUsersInvitesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteTeamUsersInvitesRequest")
		return echo.ErrBadRequest
	}

	if err := h.TeamService.DeleteTeamInvitation(c.Request().Context(), req.TeamID, req.Email); err != nil {
		if errors.Is(err, team.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to delete team invitation")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

