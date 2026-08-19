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
}

func (h *Handlers) PostRefreshToken(c hs.BaseContext) error {
	if _, err := helpers.Bind[PostRefreshTokenRequest](c); err != nil {
		c.Log.WithError(err).Debug("failed to bind PostRefreshTokenRequest")

		return echo.ErrBadRequest
	}

	cookie, err := h.CookieService.GetRefreshToken(c)
	if err != nil {
		c.Log.WithError(err).Debug("missing refresh token cookie")
		return echo.ErrUnauthorized
	}
	
	refreshTokenStr := cookie.Value

	accessToken, refreshToken, err := h.AuthenticationService.Refresh(c.Request().Context(), refreshTokenStr)
	if err != nil {
		c.Log.WithError(err).Debug("failed to refresh access and refresh token")

		return echo.ErrUnauthorized
	}

	c.Log.Info("access and refresh token refreshed")
	
	h.CookieService.SetAccessToken(c, accessToken)
	h.CookieService.SetRefreshToken(c, refreshToken)

	return c.JSON(http.StatusOK, PostRefreshTokenResponse{})
}
