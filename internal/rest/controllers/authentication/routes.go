package authentication

import (
	"github.com/labstack/echo/v4"
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
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	authenticationService authentication.Service,
	teamService team.Service,
	userService user.Service,
) {
	h := &Handlers{
		AuthenticationService: authenticationService,
		TeamService:           teamService,
		UserService:           userService,
	}

	BaseHandler := handlers.BaseHandlerFactory(logger)

	authGroup := e.Group("/auth")

	authGroup.POST("/login", BaseHandler(h.PostLogin))
	authGroup.POST("/refresh", BaseHandler(h.PostRefreshToken))
}
