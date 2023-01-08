package teams

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type PostTeamUsersInviteRequest struct {
	TeamID uint   `param:"teamId" validate:"required,numeric,gt=0"`
	Emails string `json:"emails" validate:"required,email"`
}

func (h *Handlers) PostTeamUsersInvite(ctx hs.AuthenticatedContext) error {
	_, err := helpers.Bind[PostTeamUsersInviteRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PostTeamUsersInviteRequest")

		return echo.ErrBadRequest
	}

	// TODO

	return ctx.NoContent(http.StatusNotImplemented)
}

type GetTeamUsersInviteURLRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamUsersInviteURLResponse struct {
	URL string `json:"url"`
}

func (h *Handlers) GetTeamUsersInviteURL(ctx hs.AuthenticatedContext) error {
	_, err := helpers.Bind[GetTeamUsersInviteURLRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetTeamUsersInviteURLRequest")

		return echo.ErrBadRequest
	}

	// TODO

	return ctx.NoContent(http.StatusNotImplemented)
}

type PostTeamUsersInviteAcceptRequest struct {
	TeamID          uint   `param:"teamId" validate:"required,numeric,gt=0"`
	InvitationToken string `param:"invitationToken" validate:"required"`
}

func (h *Handlers) PostTeamUsersInviteAccept(ctx hs.AuthenticatedContext) error {
	_, err := helpers.Bind[PostTeamUsersInviteAcceptRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PostTeamUsersInviteAcceptRequest")

		return echo.ErrBadRequest
	}

	// TODO

	return ctx.NoContent(http.StatusNotImplemented)
}
