package rest

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/probes"
	"github.com/opsway-io/backend/internal/rest/controllers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/oauth"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Port                 uint32 `default:"8001"`
	GZIPCompressionLevel int    `default:"5"`
}

type Server struct {
	echo   *echo.Echo
	config Config
}

func NewServer(
	conf Config,
	oauthConfig *oauth.Config,
	logger *logrus.Logger,
	authenticationService authentication.Service,
	userService user.Service,
	teamService team.Service,
	monitorService monitor.Service,
	httpResultService probes.Service,
) (*Server, error) {
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true
	e.Validator = helpers.NewValidator()

	e.Use(
		middleware.Recover(),
		middleware.Logger(),
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{"*"},
		}),
		middleware.GzipWithConfig(middleware.GzipConfig{
			Level:   conf.GZIPCompressionLevel,
			Skipper: middleware.DefaultGzipConfig.Skipper,
		}),
	)

	root := e.Group("/api/v1")

	controllers.Register(
		root,
		logger.WithField("module", "rest_controllers"),
		authenticationService,
		userService,
		teamService,
		monitorService,
		httpResultService,
	)

	if oauthConfig != nil {
		oauth.Register(
			root,
			logger.WithField("module", "rest_oauth"),
			oauthConfig,
			authenticationService,
			userService,
		)

		logger.Info("OAuth endpoints registered")
	}

	return &Server{
		echo:   e,
		config: conf,
	}, nil
}

func (s *Server) Start() error {
	return errors.Wrap(s.echo.Start(fmt.Sprintf(":%d", s.config.Port)), "Failed to start server")
}

func (s *Server) Shutdown(ctx context.Context) error {
	return errors.Wrap(s.echo.Shutdown(ctx), "Failed to shutdown server")
}
