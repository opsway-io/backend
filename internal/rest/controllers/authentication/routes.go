package authentication

import (
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	TeamService           team.Service
	UserService           user.Service
	OAuthConfig           *OAuthConfig
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	oAuthConfig *OAuthConfig,
	authenticationService authentication.Service,
	teamService team.Service,
	userService user.Service,
) {
	h := &Handlers{
		OAuthConfig:           oAuthConfig,
		AuthenticationService: authenticationService,
		TeamService:           teamService,
		UserService:           userService,
	}

	BaseHandler := handlers.BaseHandlerFactory(logger)

	authGroup := e.Group("/auth")

	authGroup.POST("/login", BaseHandler(h.PostLogin))
	authGroup.POST("/refresh", BaseHandler(h.PostRefreshToken))

	if oAuthConfig != nil {
		goth.UseProviders(
			github.New(oAuthConfig.GithubClientID, oAuthConfig.GithubClientSecret, oAuthConfig.GithubCallbackURL, []string{
				"user:email",
				"read:user",
			}...),

			google.New(oAuthConfig.GoogleClientID, oAuthConfig.GoogleClientSecret, oAuthConfig.GoogleCallbackURL, []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			}...),
		)

		oAuthGroup := e.Group("/auth/:provider")

		oAuthGroup.GET("", BaseHandler(h.GetOAuthLogin))
		oAuthGroup.GET("/callback", BaseHandler(h.GetOAuthCallback))

		logger.Info("OAuth endpoints enabled")
	}
}