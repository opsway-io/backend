package rest

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opsway-io/backend/internal/jwt"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/rest/helpers"
	v1 "github.com/opsway-io/backend/internal/rest/v1"
	"github.com/opsway-io/backend/internal/user"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Port                 uint32 `default:"8080"`
	GZIPCompressionLevel int    `default:"5"`
}

type Server struct {
	echo   *echo.Echo
	config Config
}

func NewServer(conf Config, logger *logrus.Logger, userService user.Service, jwtService jwt.Service, monitorService monitor.Service) (*Server, error) {
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true
	e.Validator = helpers.NewValidator()

	e.Use(
		middleware.Recover(),
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{"*"},
		}),
		middleware.GzipWithConfig(middleware.GzipConfig{
			Level:   conf.GZIPCompressionLevel,
			Skipper: middleware.DefaultGzipConfig.Skipper,
		}),
	)

	root := e.Group("/v1")

	v1.Register(
		root,
		logger.WithField("module", "rest"),
		userService,
		jwtService,
		monitorService,
	)

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
