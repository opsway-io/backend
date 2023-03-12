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

func (h *Handlers) PutTeamAvatar(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutTeamAvatarRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutTeamAvatarRequest")

		return echo.ErrBadRequest
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.Log.WithError(err).Debug("failed to get file from form")

		return echo.ErrBadRequest
	}

	src, err := file.Open()
	if err != nil {
		c.Log.WithError(err).Debug("failed to open file")

		return echo.ErrBadRequest
	}
	defer src.Close()

	if err = h.TeamService.UploadAvatar(
		c.Request().Context(),
		req.TeamID,
		src,
	); err != nil {
		c.Log.WithError(err).Debug("failed to update team")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type DeleteTeamAvatarRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) DeleteTeamAvatar(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteTeamAvatarRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteTeamAvatarRequest")

		return echo.ErrBadRequest
	}

	if err = h.TeamService.DeleteAvatar(
		c.Request().Context(),
		req.TeamID,
	); err != nil {
		c.Log.WithError(err).Debug("failed to delete team avatar")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
