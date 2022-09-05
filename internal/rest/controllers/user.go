package controllers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/models"
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

	fmt.Println(req) // TODO: implement

	return ctx.JSON(http.StatusNotImplemented, nil)
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

	fmt.Println(req) // TODO: implement

	return ctx.JSON(http.StatusNotImplemented, nil)
}
