package users

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
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

	AuthGuard := mw.AuthGuardFactory(logger, authenticationService)
	CurrentUserGuard := mw.CurrentUSerGuardFactory(logger)

	BaseHandler := handlers.BaseHandlerFactory(logger)
	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	e.POST("/users/:userId/password/reset", BaseHandler(h.PostUserPasswordReset))
	e.POST("/users/:userId/password/reset/new", BaseHandler(h.PostUserPasswordResetNewPassword))

	usersGroup := e.Group(
		"/users/:userId",
		AuthGuard(),
		CurrentUserGuard(),
	)

	usersGroup.GET("", AuthHandler(h.GetUser))
	usersGroup.PUT("", AuthHandler(h.PutUser))
	usersGroup.DELETE("", AuthHandler(h.DeleteUser))

	usersGroup.PUT("/password", AuthHandler(h.PutUserPassword))

	usersGroup.PUT("/avatar", AuthHandler(h.PutUserAvatar))
	usersGroup.DELETE("/avatar", AuthHandler(h.DeleteUserAvatar))
}
