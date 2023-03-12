package users

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/user"
)

type PutUserAvatarRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) PutUserAvatar(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutUserAvatarRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutUserAvatarRequest")

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

	if err := h.UserService.UploadAvatar(
		c.Request().Context(),
		req.UserID,
		src,
	); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to set user avatar")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type DeleteUserAvatarRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) DeleteUserAvatar(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteUserAvatarRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteUserAvatarRequest")

		return echo.ErrBadRequest
	}

	if err := h.UserService.DeleteAvatar(c.Request().Context(), req.UserID); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to delete user avatar")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
