package oauth

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

type Config struct {
	GithubClientID     string `mapstructure:"github_client_id"`
	GithubClientSecret string `mapstructure:"github_client_secret"`
	GithubCallbackURL  string `mapstructure:"github_callback_url"`
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	config *Config,
	authenticationService authentication.Service,
	userService user.Service,
) {
	// Setup providers

	githubScopes := []string{
		"user:email",
		"read:user",
	}

	goth.UseProviders(
		github.New(config.GithubClientID, config.GithubClientSecret, config.GithubCallbackURL, githubScopes...),
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

			return echo.ErrUnauthorized
		}

		user, err := userService.GetUserAndTeamsByEmailAddress(c.Request().Context(), gothUser.Email)
		if err != nil {
			return echo.ErrUnauthorized
		}

		accessToken, refreshToken, err := authenticationService.Generate(user)
		if err != nil {
			return echo.ErrInternalServerError
		}

		c.SetCookie(&http.Cookie{
			Name:  "access_token",
			Value: accessToken,
		})

		c.SetCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})

		return c.Redirect(http.StatusTemporaryRedirect, "/login/oauth")
	})
}
