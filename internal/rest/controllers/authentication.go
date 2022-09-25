package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/models"
)

type PostLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type PostLoginResponse struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	User         models.User `json:"user"`
}

func (h *Handlers) PostLogin(ctx hs.BaseContext) error {
	req, err := helpers.Bind[PostLoginRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PostLoginRequest")

		return echo.ErrBadRequest
	}

	user, err := h.UserService.GetByEmail(ctx.Request().Context(), req.Email)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to get user")

		return echo.ErrUnauthorized
	}

	ctx.Log = ctx.Log.WithField("user_id", user.ID)

	if ok := user.CheckPassword(req.Password); !ok {
		ctx.Log.Debug("password invalid")

		return echo.ErrUnauthorized
	}

	accessToken, refreshToken, err := h.AuthenticationService.Generate(user)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to generate access and refresh token for user")

		return echo.ErrInternalServerError
	}

	ctx.Log.Info("user authenticated")

	return ctx.JSON(http.StatusOK, PostLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         models.UserToResponse(*user),
	})
}

type PostRefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type PostRefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func (h *Handlers) PostRefreshToken(ctx hs.BaseContext) error {
	req, err := helpers.Bind[PostRefreshTokenRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PostRefreshTokenRequest")

		return echo.ErrBadRequest
	}

	accessToken, refreshToken, err := h.AuthenticationService.Refresh(req.RefreshToken)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to refresh access and refresh token")

		return echo.ErrUnauthorized
	}

	ctx.Log.Info("access and refresh token refreshed")

	return ctx.JSON(http.StatusOK, PostRefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}
