package rest

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opsway-io/backend/internal/jwt"
	v1 "github.com/opsway-io/backend/internal/rest/v1"
	"github.com/opsway-io/backend/internal/rest/validator"
	"github.com/opsway-io/backend/internal/user"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Config struct {
	Port                 uint32 `default:"8080"`
	GZIPCompressionLevel int    `default:"5"`
}

type Server struct {
	echo   *echo.Echo
	config Config
}

func NewServer(conf Config, logger *zap.Logger, userService user.Service, jwtService jwt.Service) (*Server, error) {
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true
	e.Validator = validator.New()

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

	root := e.Group("")

	v1.Register(
		root,
		logger,
		userService,
		jwtService,
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
