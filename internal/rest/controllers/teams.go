package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
)

type GetTeamRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamResponse struct {
	// TODO
}

func (h *Handlers) GetTeam(ctx hs.AuthenticatedContext) error {
	_, err := helpers.Bind[GetTeamRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetTeamRequest")

		return echo.ErrBadRequest
	}

	// TODO

	return ctx.JSON(http.StatusNotImplemented, nil)
}

type GetTeamUsersRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamUsersResponse struct {
	Users []GetTeamUsersResponseUser `json:"users"`
}

type GetTeamUsersResponseUser struct {
	ID          uint          `json:"id"`
	Name        string        `json:"name"`
	DisplayName *string       `json:"displayName"`
	Email       string        `json:"email"`
	AvatarURL   *string       `json:"avatarUrl"`
	Role        entities.Role `json:"role"`
}

func (h *Handlers) GetTeamUsers(ctx hs.AuthenticatedContext) error {
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
			DisplayName: u.DisplayName,
			Name:        u.Name,
			AvatarURL:   u.Avatar,
			Role:        u.Role,
		}
	}

	return GetTeamUsersResponse{
		Users: res,
	}
}
