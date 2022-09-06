package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/models"
	"github.com/opsway-io/backend/internal/team"
	"github.com/pkg/errors"
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

	team, err := h.TeamService.GetByID(ctx.Request().Context(), req.TeamID)
	if err != nil {
		l.WithError(err).Debug("failed to get team")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, models.TeamToResponse(*team))
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

	t := models.RequestToTeam(req.Team)
	t.ID = req.TeamID

	if err := h.TeamService.Update(ctx.Request().Context(), &t); err != nil {
		if errors.Is(err, team.ErrNotFound) {
			l.WithError(err).Debug("team not found")

			return echo.ErrNotFound
		}

		l.WithError(err).Debug("failed to update team")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, models.TeamToResponse(t))
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

	users, err := h.UserService.GetByTeamID(ctx.Request().Context(), req.TeamID)
	if err != nil {
		l.WithError(err).Debug("failed to get users")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, GetTeamUsersResponse{
		Users: models.UsersToResponse(*users),
	})
}
