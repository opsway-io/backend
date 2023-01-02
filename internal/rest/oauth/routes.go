package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

var ErrCouldNotScrapeAvatar = errors.New("could not scrape avatar")

type Config struct {
	GithubClientID     string `mapstructure:"github_client_id"`
	GithubClientSecret string `mapstructure:"github_client_secret"`
	GithubCallbackURL  string `mapstructure:"github_callback_url"`

	GoogleClientID     string `mapstructure:"google_client_id"`
	GoogleClientSecret string `mapstructure:"google_client_secret"`
	GoogleCallbackURL  string `mapstructure:"google_callback_url"`

	SuccessURL string `mapstructure:"success_url" default:"/login/oauth"`
	FailureURL string `mapstructure:"failure_url" default:"/login"`

	CookieDomain string `mapstructure:"cookie_domain"`
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	config *Config,
	authenticationService authentication.Service,
	userService user.Service,
) {
	// Setup providers

	goth.UseProviders(
		github.New(config.GithubClientID, config.GithubClientSecret, config.GithubCallbackURL, []string{
			"user:email",
			"read:user",
		}...),

		google.New(config.GoogleClientID, config.GoogleClientSecret, config.GoogleCallbackURL, []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		}...),
	)

	// Routes

	g := e.Group("/auth/:provider")

	g.GET("", func(c echo.Context) error {
		req := c.Request().WithContext(context.WithValue(c.Request().Context(), gothic.ProviderParamKey, c.Param("provider")))

		gothic.BeginAuthHandler(c.Response(), req)

		return nil
	})

	g.GET("/callback", func(c echo.Context) error {
		gothUser, err := gothic.CompleteUserAuth(c.Response(), c.Request())
		if err != nil {
			logger.WithError(err).Error("failed to complete user oauth flow")

			return c.Redirect(http.StatusTemporaryRedirect, config.FailureURL)
		}

		user, err := getOrCreateUser(c, userService, gothUser)
		if err != nil {
			logger.WithError(err).Error("failed to get or create user to complete oauth flow")

			return c.Redirect(http.StatusTemporaryRedirect, config.FailureURL)
		}

		accessToken, refreshToken, err := authenticationService.Generate(c.Request().Context(), user)
		if err != nil {
			logger.WithError(err).Error("failed to generate access token to complete oauth flow")

			return c.Redirect(http.StatusTemporaryRedirect, config.FailureURL)
		}

		c.SetCookie(&http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/",
			Domain:   config.CookieDomain,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(1 * time.Minute),
		})

		c.SetCookie(&http.Cookie{
			Name:     "access_token",
			Value:    accessToken,
			Domain:   config.CookieDomain,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(1 * time.Minute),
		})

		if len(user.Teams) > 0 {
			c.SetCookie(&http.Cookie{
				Name:     "team_id",
				Value:    fmt.Sprintf("%d", user.Teams[0].ID),
				Path:     "/",
				Domain:   config.CookieDomain,
				SameSite: http.SameSiteLaxMode,
				Expires:  time.Now().Add(1 * time.Minute),
			})
		}

		return c.Redirect(http.StatusTemporaryRedirect, config.SuccessURL)
	})
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
		if err := userService.ScrapeUserAvatarFromURL(c.Request().Context(), u.ID, gothUser.AvatarURL); err != nil {
			return nil, err
		}
	}

	return u, nil
}
