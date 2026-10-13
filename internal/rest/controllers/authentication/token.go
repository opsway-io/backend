package authentication

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type PostRefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type PostRefreshTokenResponse struct {
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

func (h *Handlers) PostRefreshToken(c hs.BaseContext) error {
	req, _ := helpers.Bind[PostRefreshTokenRequest](c)

	var refreshTokenStr string
	if cookie, err := h.CookieService.GetRefreshToken(c); err == nil && cookie.Value != "" {
		refreshTokenStr = cookie.Value
	} else if req != nil && req.RefreshToken != "" {
		refreshTokenStr = req.RefreshToken
	}

	if refreshTokenStr == "" {
		c.Log.Debug("missing refresh token in cookie or request body")
		return echo.ErrUnauthorized
	}

	accessToken, refreshToken, err := h.AuthenticationService.Refresh(c.Request().Context(), refreshTokenStr)
	if err != nil {
		c.Log.WithError(err).Debug("failed to refresh access and refresh token")

		return echo.ErrUnauthorized
	}

	c.Log.Info("access and refresh token refreshed")

	h.CookieService.SetAccessToken(c, accessToken)
	h.CookieService.SetRefreshToken(c, refreshToken)

	return c.JSON(http.StatusOK, PostRefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}
