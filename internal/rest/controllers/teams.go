package controllers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"k8s.io/utils/pointer"
)

type GetTeamRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	DisplayName *string   `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (h *Handlers) GetTeam(ctx hs.AuthenticatedContext) error {
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

	return ctx.JSON(http.StatusOK, newGetTeamResponse(team))
}

func newGetTeamResponse(t *entities.Team) GetTeamResponse {
	return GetTeamResponse{
		ID:          t.ID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		AvatarURL:   nil, // TODO
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

type GetTeamUsersRequest struct {
	TeamID uint    `param:"teamId" validate:"required,numeric,gt=0"`
	Offset *int    `query:"offset" validate:"numeric,gte=0" default:"0"`
	Limit  *int    `query:"limit" validate:"numeric,gt=0" default:"10"`
	Query  *string `query:"query" validate:"omitempty,min=3"`
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

	users, err := h.TeamService.GetUsersByID(ctx.Request().Context(), req.TeamID, req.Offset, req.Limit, req.Query)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to get users")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, newGetTeamUsersResponse(users, h.UserService))
}

func newGetTeamUsersResponse(users *[]team.TeamUser, userService user.Service) GetTeamUsersResponse {
	res := make([]GetTeamUsersResponseUser, len(*users))

	for i, u := range *users {
		res[i] = GetTeamUsersResponseUser{
			ID:          u.ID,
			Email:       u.Email,
			DisplayName: u.DisplayName,
			Name:        u.Name,
			Role:        u.Role,
		}

		if u.HasAvatar {
			res[i].AvatarURL = pointer.String(userService.GetUserAvatarURLByID(u.ID))
		}
	}

	return GetTeamUsersResponse{
		Users: res,
	}
}
