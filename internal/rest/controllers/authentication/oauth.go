package authentication

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/user"
)

var ErrCouldNotScrapeAvatar = errors.New("could not scrape avatar")

type OAuthConfig struct {
	GithubClientID     string `mapstructure:"github_client_id"`
	GithubClientSecret string `mapstructure:"github_client_secret"`
	GithubCallbackURL  string `mapstructure:"github_callback_url"`

	GoogleClientID     string `mapstructure:"google_client_id"`
	GoogleClientSecret string `mapstructure:"google_client_secret"`
	GoogleCallbackURL  string `mapstructure:"google_callback_url"`

	SuccessURL string `mapstructure:"success_url" default:"/login/oauth"`
	FailureURL string `mapstructure:"failure_url" default:"/login"`
}

func (h *Handlers) GetOAuthLogin(c hs.BaseContext) error {
	req := c.Request().WithContext(context.WithValue(c.Request().Context(), gothic.ProviderParamKey, c.Param("provider")))

	gothic.BeginAuthHandler(c.Response(), req)

	return nil
}

func (h *Handlers) GetOAuthCallback(c hs.BaseContext) error {
	gothUser, err := gothic.CompleteUserAuth(c.Response(), c.Request())
	if err != nil {
		c.Log.WithError(err).Error("failed to complete user oauth flow")

		return c.Redirect(http.StatusTemporaryRedirect, h.OAuthConfig.FailureURL)
	}

	user, err := getOrCreateUser(c, h.UserService, gothUser)
	if err != nil {
		c.Log.WithError(err).Error("failed to get or create user to complete oauth flow")

		return c.Redirect(http.StatusTemporaryRedirect, h.OAuthConfig.FailureURL)
	}

	accessToken, refreshToken, err := h.AuthenticationService.Generate(c.Request().Context(), user)
	if err != nil {
		c.Log.WithError(err).Error("failed to generate access token to complete oauth flow")

		return c.Redirect(http.StatusTemporaryRedirect, h.OAuthConfig.FailureURL)
	}

	h.CookieService.SetRefreshToken(c, refreshToken)
	h.CookieService.SetAccessToken(c, accessToken)

	return c.Redirect(http.StatusTemporaryRedirect, h.OAuthConfig.SuccessURL)
}

func getOrCreateUser(c echo.Context, userService user.Service, gothUser goth.User) (*entities.User, error) {
	u, err := userService.GetUserAndTeamsByEmailAddress(c.Request().Context(), gothUser.Email)
	if err != nil && !errors.Is(err, user.ErrNotFound) {
		return nil, err
	}

	// User already exists, return it

	if u != nil {
		return u, nil
	}

	// User does not exist, create it

	u = &entities.User{
		Name:        gothUser.Name,
		DisplayName: &gothUser.NickName,
	}

	u.SetEmail(gothUser.Email)

	if err := userService.Create(c.Request().Context(), u); err != nil {
		return nil, err
	}

	if gothUser.AvatarURL != "" {
		if err := userService.SetAvatarFromURL(c.Request().Context(), u.ID, gothUser.AvatarURL); err != nil {
			return nil, err
		}
	}

	return u, nil
}
