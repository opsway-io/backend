package controllers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/user"
	"github.com/pkg/errors"
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

func (h *Handlers) GetUser(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetUserRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetUserRequest")

		return echo.ErrBadRequest
	}

	u, err := h.UserService.GetUserAndTeamsByUserID(ctx.Request().Context(), req.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to get user")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, newGetUserResponse(u, h.UserService))
}

func newGetUserResponse(u *entities.User, userService user.Service) GetUserResponse {
	teams := make([]GetUserResponseTeam, len(u.Teams))

	for i, t := range u.Teams {
		teams[i] = GetUserResponseTeam{
			ID:          t.ID,
			Name:        t.Name,
			DisplayName: t.DisplayName,
			AvatarURL:   nil, // TODO
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
		res.AvatarURL = pointer.StringPtr(userService.GetUserAvatarURLByID(u.ID))
	}

	return res
}

type PutUserRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
	// TODO
}

type PutUserResponse struct {
	// TODO
}

func (h *Handlers) PutUser(ctx hs.AuthenticatedContext) error {
	_, err := helpers.Bind[PutUserRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutUserRequest")

		return echo.ErrBadRequest
	}

	// TODO

	return ctx.NoContent(http.StatusNotImplemented)
}

func (h *Handlers) DeleteUser(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetUserRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetUserRequest")

		return echo.ErrBadRequest
	}

	if err := h.UserService.Delete(ctx.Request().Context(), req.UserID); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to delete user")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusOK)
}
