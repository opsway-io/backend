package teams

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"k8s.io/utils/pointer"
)

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
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	DisplayName *string           `json:"displayName"`
	Email       string            `json:"email"`
	AvatarURL   *string           `json:"avatarUrl"`
	Role        entities.TeamRole `json:"role"`
}

func (h *Handlers) GetTeamUsers(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetTeamUsersRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetTeamUsersRequest")

		return echo.ErrBadRequest
	}

	users, err := h.TeamService.GetUsersByID(c.Request().Context(), req.TeamID, req.Offset, req.Limit, req.Query)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get users")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, newGetTeamUsersResponse(users, h.UserService))
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

type DeleteTeamUserRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) DeleteTeamUser(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteTeamUserRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteTeamUserRequest")

		return echo.ErrBadRequest
	}

	if err = h.TeamService.RemoveUser(
		c.Request().Context(),
		req.TeamID,
		req.UserID,
	); err != nil {
		c.Log.WithError(err).Debug("failed to delete team user")

		if errors.Is(err, team.ErrUserNotFound) {
			return echo.ErrNotFound
		}

		if errors.Is(err, team.ErrCannotRemoveOwner) {
			return echo.ErrForbidden
		}

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PutTeamUserRequest struct {
	TeamID uint              `param:"teamId" validate:"required,numeric,gt=0"`
	UserID uint              `param:"userId" validate:"required,numeric,gt=0"`
	Role   entities.TeamRole `json:"role" validate:"required,teamRole"`
}

func (h *Handlers) PutTeamUser(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutTeamUserRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutTeamUserRequest")

		return echo.ErrBadRequest
	}

	if req.Role == entities.TeamRoleOwner && c.TeamRole != nil && *c.TeamRole != entities.TeamRoleOwner {
		c.Log.Debug("Non-team owner not allowed to make other users owner")

		return echo.ErrForbidden
	}

	if err = h.TeamService.UpdateUserRole(
		c.Request().Context(),
		req.TeamID,
		req.UserID,
		req.Role,
	); err != nil {
		if errors.Is(err, team.ErrUserNotFound) {
			c.Log.WithError(err).Debug("failed to update team user")

			return echo.ErrNotFound
		}

		c.Log.WithError(err).Debug("failed to update team user")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
