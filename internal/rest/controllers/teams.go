package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/models"
	"github.com/opsway-io/backend/internal/team"
	"github.com/pkg/errors"
)

type GetTeamRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamResponse struct {
	models.Team
}

func (h *Handlers) GetTeam(ctx handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetTeamRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetTeamRequest")

		return echo.ErrBadRequest
	}

	team, err := h.TeamService.GetByID(ctx.Request().Context(), req.TeamID)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to get team")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, models.TeamToResponse(*team))
}

type PutTeamRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
	models.Team
}

type PutTeamResponse struct {
	models.Team
}

func (h *Handlers) PutTeam(ctx handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PutTeamRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutTeamRequest")

		return echo.ErrBadRequest
	}

	t := models.RequestToTeam(req.Team)
	t.ID = req.TeamID

	if err := h.TeamService.Update(ctx.Request().Context(), &t); err != nil {
		if errors.Is(err, team.ErrNotFound) {
			ctx.Log.WithError(err).Debug("team not found")

			return echo.ErrNotFound
		}
		if errors.Is(err, entities.ErrIllegalTeamNameFormat) {
			ctx.Log.WithError(err).Debug("illegal team name format")

			return echo.ErrBadRequest
		}

		ctx.Log.WithError(err).Debug("failed to update team")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, models.TeamToResponse(t))
}

type GetTeamUsersRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamUsersResponse struct {
	Users []GetTeamUsersResponseUser `json:"users"`
}

type GetTeamUsersResponseUser struct {
	ID          uint          `json:"id"`
	Email       string        `json:"email"`
	DisplayName string        `json:"displayName"`
	Name        string        `json:"name"`
	Picture     string        `json:"picture"`
	Role        entities.Role `json:"role"`
}

func (h *Handlers) GetTeamUsers(ctx handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetTeamUsersRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetTeamUsersRequest")

		return echo.ErrBadRequest
	}

	users, err := h.TeamService.GetUsersByID(ctx.Request().Context(), req.TeamID)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to get users")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, newGetTeamUsersResponse(users))
}

func newGetTeamUsersResponse(users *[]team.TeamUser) GetTeamUsersResponse {
	res := make([]GetTeamUsersResponseUser, len(*users))

	for i, u := range *users {
		res[i] = GetTeamUsersResponseUser{
			ID:          u.ID,
			Email:       u.Email,
			DisplayName: *u.DisplayName,
			Name:        u.Name,
			// Picture:     u.Picture,
			Role: u.Role,
		}
	}

	return GetTeamUsersResponse{
		Users: res,
	}
}
