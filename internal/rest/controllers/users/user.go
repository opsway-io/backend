package users

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

type GetUserRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
}

type GetUserResponse struct {
	ID          uint                  `json:"id"`
	Name        string                `json:"name"`
	DisplayName *string               `json:"displayName"`
	Email       string                `json:"email"`
	AvatarURL   *string               `json:"avatarUrl"`
	Teams       []GetUserResponseTeam `json:"teams"`
	CreatedAt   time.Time             `json:"createdAt"`
	UpdatedAt   time.Time             `json:"updatedAt"`
}

type GetUserResponseTeam struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	DisplayName *string `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl"`
}

func (h *Handlers) GetUser(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetUserRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetUserRequest")

		return echo.ErrBadRequest
	}

	u, err := h.UserService.GetUserAndTeamsByUserID(c.Request().Context(), req.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to get user")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, newGetUserResponse(u, h.UserService, h.TeamService))
}

func newGetUserResponse(u *entities.User, userService user.Service, teamService team.Service) GetUserResponse {
	teams := make([]GetUserResponseTeam, len(u.Teams))

	for i, t := range u.Teams {
		teams[i] = GetUserResponseTeam{
			ID:          t.ID,
			Name:        t.Name,
			DisplayName: t.DisplayName,
		}

		if t.HasAvatar {
			teams[i].AvatarURL = pointer.StringPtr(teamService.GetAvatarURLByID(t.ID))
		}
	}

	res := GetUserResponse{
		ID:          u.ID,
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Email:       u.Email,
		Teams:       teams,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}

	if u.HasAvatar {
		res.AvatarURL = pointer.StringPtr(userService.GetAvatarURLByID(u.ID))
	}

	return res
}

type PutUserRequest struct {
	UserID      uint   `param:"userId" validate:"required,numeric,gt=0"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	DisplayName string `json:"displayName" validate:"required,min=0,max=255"`
	Email       string `json:"email" validate:"required,email"`
}

func (h *Handlers) PutUser(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutUserRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutUserRequest")

		return echo.ErrBadRequest
	}

	u := &entities.User{
		ID:          req.UserID,
		Name:        req.Name,
		DisplayName: &req.DisplayName,
	}
	u.SetEmail(req.Email)

	if err := h.UserService.Update(c.Request().Context(), u); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to update user")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handlers) DeleteUser(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetUserRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetUserRequest")

		return echo.ErrBadRequest
	}

	if err := h.UserService.Delete(c.Request().Context(), req.UserID); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to delete user")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
