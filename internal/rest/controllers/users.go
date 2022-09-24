package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/models"
	"github.com/opsway-io/backend/internal/user"
	"github.com/pkg/errors"
)

type GetUserRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
}

type GetUserResponse struct {
	models.User
}

func (h *Handlers) GetUser(ctx handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetUserRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetUserRequest")

		return echo.ErrBadRequest
	}

	u, err := h.UserService.GetByID(ctx.Request().Context(), req.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to get user")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, models.UserToResponse(*u))
}

type PutUserRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
	models.User
}

type PutUserResponse struct {
	models.User
}

func (h *Handlers) PutUser(ctx handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PutUserRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutUserRequest")

		return echo.ErrBadRequest
	}

	u := models.RequestToUser(req.User)
	u.ID = req.UserID

	if err := h.UserService.Update(ctx.Request().Context(), &u); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to update user")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusOK)
}

func (h *Handlers) DeleteUser(ctx handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetUserRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetUserRequest")

		return echo.ErrBadRequest
	}

	if err := h.UserService.Delete(ctx.Request().Context(), req.UserID); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to delete user")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusOK)
}
