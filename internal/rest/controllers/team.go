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

type GetTeamRequest struct {
	TeamID int `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamResponse struct {
	models.Team
}

func (h *Handlers) GetTeam(ctx handlers.AuthenticatedContext, l *logrus.Entry) error {
	req, err := helpers.Bind[GetTeamRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind GetTeamRequest")

		return echo.ErrBadRequest
	}

	fmt.Println(req) // TODO: implement

	return ctx.JSON(http.StatusNotImplemented, nil)
}

type PutTeamRequest struct {
	TeamID int `param:"teamId" validate:"required,numeric,gt=0"`
	models.Team
}

type PutTeamResponse struct {
	models.Team
}

func (h *Handlers) PutTeam(ctx handlers.AuthenticatedContext, l *logrus.Entry) error {
	req, err := helpers.Bind[PutTeamRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind PutTeamRequest")

		return echo.ErrBadRequest
	}

	fmt.Println(req) // TODO: implement

	return ctx.JSON(http.StatusNotImplemented, nil)
}

type GetTeamUsersRequest struct {
	TeamID int `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamUsersResponse struct {
	Users []models.User `json:"users"`
}

func (h *Handlers) GetTeamUsers(ctx handlers.AuthenticatedContext, l *logrus.Entry) error {
	req, err := helpers.Bind[GetTeamUsersRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind GetTeamUsersRequest")

		return echo.ErrBadRequest
	}

	fmt.Println(req) // TODO: implement

	return ctx.JSON(http.StatusNotImplemented, nil)
}
