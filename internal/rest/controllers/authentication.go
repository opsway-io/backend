package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

type User struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type PostLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type PostLoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	User         User   `json:"user"`
}

func (h *Handlers) PostLogin(ctx echo.Context, l *logrus.Entry) error {
	req, err := helpers.Bind[PostLoginRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind PostLoginRequest")

		return echo.ErrBadRequest
	}

	user, err := h.UserService.GetUserByEmail(ctx.Request().Context(), req.Email)
	if err != nil {
		l.WithError(err).Debug("failed to get user")

		return echo.ErrUnauthorized
	}

	l = l.WithField("user_id", user.ID)

	if ok := user.CheckPassword(req.Password); !ok {
		l.Debug("password invalid")

		return echo.ErrUnauthorized
	}

	token, refresh, err := h.JWTService.Generate(user)
	if err != nil {
		l.WithError(err).Debug("failed to generate token for user")

		return echo.ErrInternalServerError
	}

	l.Info("user authenticated")

	return ctx.JSON(http.StatusOK, PostLoginResponse{
		Token:        token,
		RefreshToken: refresh,
		User:         userToResponse(user),
	})
}

type PostRefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type PostRefreshResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

func (h *Handlers) PostRefresh(ctx echo.Context, l *logrus.Entry) error {
	req, err := helpers.Bind[PostRefreshRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind PostLoginRequest")

		return echo.ErrBadRequest
	}

	token, refresh, err := h.JWTService.Refresh(req.RefreshToken)
	if err != nil {
		l.WithError(err).Debug("failed to renew token")

		return echo.ErrUnauthorized
	}

	l.Info("token refreshed")

	return ctx.JSON(http.StatusOK, PostRefreshResponse{
		Token:        token,
		RefreshToken: refresh,
	})
}

func userToResponse(user *user.User) User {
	return User{
		ID:          user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		CreatedAt:   user.CreatedAt.Unix(),
		UpdatedAt:   user.UpdatedAt.Unix(),
	}
}
