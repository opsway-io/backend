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
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func (h *Handlers) PostRefreshToken(c hs.BaseContext) error {
	req, err := helpers.Bind[PostRefreshTokenRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostRefreshTokenRequest")

		return echo.ErrBadRequest
	}

	// Fall back to the refresh token cookie if none is given in the body
	if req.RefreshToken == "" {
		cookie, err := h.CookieService.GetRefreshToken(c)
		if err != nil || cookie.Value == "" {
			c.Log.Debug("missing refresh token")

			return echo.ErrUnauthorized
		}

		req.RefreshToken = cookie.Value
	}

	accessToken, refreshToken, err := h.AuthenticationService.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		c.Log.WithError(err).Debug("failed to refresh access and refresh token")

		return echo.ErrUnauthorized
	}

	h.CookieService.SetAccessToken(c, accessToken)
	h.CookieService.SetRefreshToken(c, refreshToken)

	c.Log.Info("access and refresh token refreshed")

	return c.JSON(http.StatusOK, PostRefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *Handlers) PostLogout(c hs.BaseContext) error {
	h.CookieService.ClearTokens(c)

	return c.NoContent(http.StatusNoContent)
}
