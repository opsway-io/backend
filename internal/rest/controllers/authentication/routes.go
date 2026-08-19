package authentication

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type Handlers struct {
	CookieService         helpers.CookieService
	AuthenticationService authentication.Service
	TeamService           team.Service
	UserService           user.Service
	OAuthConfig           *OAuthConfig
	AuthConfig            *authentication.Config
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	cookieService helpers.CookieService,
	oAuthConfig *OAuthConfig,
	AuthConfig *authentication.Config,
	authenticationService authentication.Service,
	teamService team.Service,
	userService user.Service,
) {
	h := &Handlers{
		CookieService:         cookieService,
		OAuthConfig:           oAuthConfig,
		AuthenticationService: authenticationService,
		AuthConfig:            AuthConfig,
		TeamService:           teamService,
		UserService:           userService,
	}

	BaseHandler := handlers.BaseHandlerFactory(logger)

	authGroup := e.Group("/auth")
	authGroup.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(10),
				Burst:     30,
				ExpiresIn: 3 * 60 * time.Second,
			},
		),
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
	}))

	authGroup.POST("/login", BaseHandler(h.PostLogin))
	authGroup.POST("/register", BaseHandler(h.PostRegister))
	authGroup.POST("/logout", BaseHandler(h.PostLogout))
	authGroup.POST("/refresh", BaseHandler(h.PostRefreshToken))
	authGroup.POST("/forgot-password", BaseHandler(h.PostForgotPassword))
	authGroup.POST("/reset-password", BaseHandler(h.PostResetPassword))

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
