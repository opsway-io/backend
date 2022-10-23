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
	UserID      uint   `param:"userId" validate:"required,numeric,gt=0"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	DisplayName string `json:"displayName" validate:"required,min=0,max=255"`
	Email       string `json:"email" validate:"required,email"`
}

func (h *Handlers) PutUser(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutUserRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutUserRequest")

		return echo.ErrBadRequest
	}

	u := &entities.User{
		ID:          req.UserID,
		Name:        req.Name,
		DisplayName: &req.DisplayName,
	}
	u.SetEmail(req.Email)

	if err := h.UserService.Update(ctx.Request().Context(), u); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to update user")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
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

	return ctx.NoContent(http.StatusNoContent)
}

type PutUserPasswordRequest struct {
	UserID      uint   `param:"userId" validate:"required,numeric,gt=0"`
	OldPassword string `json:"oldPassword" validate:"required,min=8,max=255"`
	NewPassword string `json:"newPassword" validate:"required,min=8,max=255"`
}

func (h *Handlers) PutUserPassword(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutUserPasswordRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutUserPasswordRequest")

		return echo.ErrBadRequest
	}

	if err := h.UserService.ChangePassword(
		ctx.Request().Context(),
		req.UserID,
		req.OldPassword,
		req.NewPassword,
	); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		if errors.Is(err, user.ErrInvalidPassword) {
			ctx.Log.WithError(err).Debug("invalid password")

			return echo.ErrUnauthorized
		}

		ctx.Log.WithError(err).Error("failed to change user password")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}

type PutUserAvatarRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) PutUserAvatar(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutUserAvatarRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutUserAvatarRequest")

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

	if err := h.UserService.UploadUserAvatar(ctx.Request().Context(), req.UserID, src); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to set user avatar")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}

type DeleteUserAvatarRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) DeleteUserAvatar(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteUserAvatarRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind DeleteUserAvatarRequest")

		return echo.ErrBadRequest
	}

	if err := h.UserService.DeleteUserAvatar(ctx.Request().Context(), req.UserID); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.Log.WithError(err).Debug("user not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to delete user avatar")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}
