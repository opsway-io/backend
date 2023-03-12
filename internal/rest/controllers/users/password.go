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

func (h *Handlers) PutUserPassword(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutUserPasswordRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutUserPasswordRequest")

		return echo.ErrBadRequest
	}

	if err := h.UserService.ChangePasswordWithOldPassword(
		c.Request().Context(),
		req.UserID,
		req.OldPassword,
		req.NewPassword,
	); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		if errors.Is(err, user.ErrInvalidPassword) {
			c.Log.WithError(err).Debug("invalid password")

			return echo.ErrBadRequest
		}

		c.Log.WithError(err).Error("failed to change user password")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PostUserPasswordResetRequest struct {
	UserID uint   `param:"userId" validate:"required,numeric,gt=0"`
	Email  string `json:"email" validate:"required,email,max=255"`
}

func (h *Handlers) PostUserPasswordReset(c hs.BaseContext) error {
	req, err := helpers.Bind[PostUserPasswordResetRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostUserPasswordResetRequest")

		return echo.ErrBadRequest
	}

	if err := h.UserService.RequestPasswordReset(
		c.Request().Context(),
		req.UserID,
	); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.Log.WithError(err).Debug("user not found")

			// We don't want to leak information about the existence of a user
			return c.NoContent(http.StatusNoContent)
		}

		c.Log.WithError(err).Error("failed to request password reset")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PostUserPasswordResetNewPasswordRequest struct {
	UserID      uint   `param:"userId" validate:"required,numeric,gt=0"`
	NewPassword string `json:"newPassword" validate:"required,min=8,max=255"`
	ResetToken  string `json:"resetToken" validate:"required,max=255"`
}

func (h *Handlers) PostUserPasswordResetNewPassword(c hs.BaseContext) error {
	req, err := helpers.Bind[PostUserPasswordResetNewPasswordRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostUserPasswordResetNewPasswordRequest")

		return echo.ErrBadRequest
	}

	if err := h.UserService.ChangePasswordWithResetToken(
		c.Request().Context(),
		req.UserID,
		req.NewPassword,
		req.ResetToken,
	); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.Log.WithError(err).Debug("user or reset token not found")

			// We don't want to leak information about the existence of a user or reset token
			return echo.ErrForbidden
		}

		c.Log.WithError(err).Error("failed to change user password with reset token")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
