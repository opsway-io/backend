package authentication

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/user"
)

type PostRegisterRequest struct {
	Name     string `json:"name" validate:"required,max=255"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

func (h *Handlers) PostRegister(c hs.BaseContext) error {
	req, err := helpers.Bind[PostRegisterRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostRegisterRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	if _, err := h.UserService.GetUserAndTeamsByEmailAddress(ctx, req.Email); err == nil {
		return echo.NewHTTPError(http.StatusConflict, "email is already registered")
	} else if !errors.Is(err, user.ErrNotFound) {
		c.Log.WithError(err).Error("failed to look up user by email")

		return echo.ErrInternalServerError
	}

	u := &entities.User{Name: req.Name}
	u.SetEmail(req.Email)

	if err := u.SetPassword(req.Password); err != nil {
		c.Log.WithError(err).Error("failed to hash password")

		return echo.ErrInternalServerError
	}

	if err := h.UserService.Create(ctx, u); err != nil {
		// The user is stored before the welcome email/event, so only fail if it was not persisted
		if u.ID == 0 {
			c.Log.WithError(err).Error("failed to create user")

			return echo.ErrInternalServerError
		}

		c.Log.WithError(err).Warn("user created, but post-create actions failed")
	}

	c.Log = c.Log.WithField("user_id", u.ID)

	accessToken, refreshToken, err := h.AuthenticationService.Generate(ctx, u)
	if err != nil {
		c.Log.WithError(err).Error("failed to generate tokens for new user")

		return echo.ErrInternalServerError
	}

	h.CookieService.SetAccessToken(c, accessToken)
	h.CookieService.SetRefreshToken(c, refreshToken)

	c.Log.Info("user registered")

	return c.JSON(http.StatusCreated, newPostLoginResponse(u, accessToken, refreshToken, h.UserService, h.TeamService))
}

type PostForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func (h *Handlers) PostForgotPassword(c hs.BaseContext) error {
	req, err := helpers.Bind[PostForgotPasswordRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	if err := h.UserService.RequestPasswordResetByEmail(c.Request().Context(), req.Email); err != nil {
		// Never reveal whether an account exists
		if !errors.Is(err, user.ErrNotFound) {
			c.Log.WithError(err).Error("failed to request password reset")
		}
	}

	return c.NoContent(http.StatusNoContent)
}

type PostResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

func (h *Handlers) PostResetPassword(c hs.BaseContext) error {
	req, err := helpers.Bind[PostResetPasswordRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	if err := h.UserService.ResetPasswordWithToken(c.Request().Context(), req.Token, req.Password); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired reset token")
		}

		c.Log.WithError(err).Error("failed to reset password")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
