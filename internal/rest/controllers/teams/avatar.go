package teams

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type PutTeamAvatarRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) PutTeamAvatar(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutTeamAvatarRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutTeamAvatarRequest")

		return echo.ErrBadRequest
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to get file from form")

		return echo.ErrBadRequest
	}

	src, err := file.Open()
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to open file")

		return echo.ErrBadRequest
	}
	defer src.Close()

	if err = h.TeamService.UploadAvatar(
		ctx.Request().Context(),
		req.TeamID,
		src,
	); err != nil {
		ctx.Log.WithError(err).Debug("failed to update team")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}

type DeleteTeamAvatarRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) DeleteTeamAvatar(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteTeamAvatarRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind DeleteTeamAvatarRequest")

		return echo.ErrBadRequest
	}

	if err = h.TeamService.DeleteAvatar(
		ctx.Request().Context(),
		req.TeamID,
	); err != nil {
		ctx.Log.WithError(err).Debug("failed to delete team avatar")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}
