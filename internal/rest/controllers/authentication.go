package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/models"
	"github.com/sirupsen/logrus"
)

type PostLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type PostLoginResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refreshToken"`
	User         models.User `json:"user"`
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

	token, err := h.JWTService.Generate(user)
	if err != nil {
		l.WithError(err).Debug("failed to generate token for user")

		return echo.ErrInternalServerError
	}

	l.Info("user authenticated")

	return ctx.JSON(http.StatusOK, PostLoginResponse{
		Token: token,
		User:  models.UserToResponse(*user),
	})
}
