package controllers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/models"
	"github.com/opsway-io/backend/internal/user"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type GetUserRequest struct {
	UserID int `param:"userId" validate:"required,numeric,gt=0"`
}

type GetUserResponse struct {
	models.User
}

func (h *Handlers) GetUser(ctx handlers.AuthenticatedContext, l *logrus.Entry) error {
	req, err := helpers.Bind[GetUserRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind GetUserRequest")

		return echo.ErrBadRequest
	}

	// TODO: Check if user is in the same team as the authenticated user

	u, err := h.UserService.GetByID(ctx.Request().Context(), req.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			l.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		l.WithError(err).Error("failed to get user")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, models.UserToResponse(*u))
}

type PutUserRequest struct {
	UserID int `param:"userId" validate:"required,numeric,gt=0"`
	models.User
}

type PutUserResponse struct {
	models.User
}

func (h *Handlers) PutUser(ctx handlers.AuthenticatedContext, l *logrus.Entry) error {
	req, err := helpers.Bind[PutUserRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind PutUserRequest")

		return echo.ErrBadRequest
	}

	if fmt.Sprint(req.UserID) != ctx.Claims.Subject {
		l.WithField("user_id", req.UserID).Debug("user id in request does not match authenticated user id")

		return echo.ErrForbidden
	}

	u := models.RequestToUser(req.User)
	if err := h.UserService.Update(ctx.Request().Context(), &u); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			l.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		l.WithError(err).Error("failed to update user")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusNotImplemented, nil)
}
