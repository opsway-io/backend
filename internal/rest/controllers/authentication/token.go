package authentication

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type PostRefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type PostRefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func (h *Handlers) PostRefreshToken(c hs.BaseContext) error {
	req, err := helpers.Bind[PostRefreshTokenRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostRefreshTokenRequest")

		return echo.ErrBadRequest
	}

	accessToken, refreshToken, err := h.AuthenticationService.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		c.Log.WithError(err).Debug("failed to refresh access and refresh token")

		return echo.ErrUnauthorized
	}

	c.Log.Info("access and refresh token refreshed")

	return c.JSON(http.StatusOK, PostRefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}
