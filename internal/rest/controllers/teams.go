package controllers

import (
	"errors"
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

	return ctx.JSON(http.StatusOK, newGetTeamResponse(team, h.TeamService))
}

func newGetTeamResponse(t *entities.Team, teamService team.Service) GetTeamResponse {
	team := GetTeamResponse{
		ID:          t.ID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}

	if t.HasAvatar {
		team.AvatarURL = pointer.StringPtr(teamService.GetAvatarURLByID(t.ID))
	}

	return team
}

type GetTeamUsersRequest struct {
	TeamID uint    `param:"teamId" validate:"required,numeric,gt=0"`
	Offset *int    `query:"offset" validate:"numeric,gte=0" default:"0"`
	Limit  *int    `query:"limit" validate:"numeric,gt=0" default:"10"`
	Query  *string `query:"query" validate:"omitempty"`
}

type GetTeamUsersResponse struct {
	Users      []GetTeamUsersResponseUser `json:"users"`
	TotalCount int                        `json:"totalCount"`
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
			res[i].AvatarURL = pointer.String(userService.GetAvatarURLByID(u.ID))
		}
	}

	totalCount := 0
	if len(*users) > 0 {
		totalCount = (*users)[0].TotalCount
	}

	return GetTeamUsersResponse{
		Users:      res,
		TotalCount: totalCount,
	}
}

type PutTeamRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gt=0"`
	DisplayName string `json:"displayName" validate:"max=255"`
}

func (h *Handlers) PutTeam(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutTeamRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutTeamRequest")

		return echo.ErrBadRequest
	}

	if err = h.TeamService.UpdateDisplayName(
		ctx.Request().Context(),
		req.TeamID,
		req.DisplayName,
	); err != nil {
		ctx.Log.WithError(err).Debug("failed to update team")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}

type PutTeamAvatarRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) PutTeamAvatar(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutTeamAvatarRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutTeamAvatarRequest")

		return echo.ErrBadRequest
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to get file from form")

		return echo.ErrBadRequest
	}

	src, err := file.Open()
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to open file")

		return echo.ErrBadRequest
	}
	defer src.Close()

	if err = h.TeamService.UploadAvatar(
		ctx.Request().Context(),
		req.TeamID,
		src,
	); err != nil {
		ctx.Log.WithError(err).Debug("failed to update team")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}

type DeleteTeamAvatarRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) DeleteTeamAvatar(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteTeamAvatarRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind DeleteTeamAvatarRequest")

		return echo.ErrBadRequest
	}

	if err = h.TeamService.DeleteAvatar(
		ctx.Request().Context(),
		req.TeamID,
	); err != nil {
		if errors.Is(err, team.ErrNotFound) {
			ctx.Log.WithError(err).Debug("team not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Debug("failed to delete team avatar")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}
