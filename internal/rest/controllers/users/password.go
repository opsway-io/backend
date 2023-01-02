package users

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/user"
)

type PutUserPasswordRequest struct {
	UserID      uint   `param:"userId" validate:"required,numeric,gt=0"`
	OldPassword string `json:"oldPassword" validate:"required,max=255"`
	NewPassword string `json:"newPassword" validate:"required,min=8,max=255"`
}

func (h *Handlers) PutUserPassword(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutUserPasswordRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutUserPasswordRequest")

		return echo.ErrBadRequest
	}

	if err := h.UserService.ChangePassword(
		ctx.Request().Context(),
		req.UserID,
		req.OldPassword,
		req.NewPassword,
	); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		if errors.Is(err, user.ErrInvalidPassword) {
			ctx.Log.WithError(err).Debug("invalid password")

			return echo.ErrBadRequest
		}

		ctx.Log.WithError(err).Error("failed to change user password")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}
