package teams

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
	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	teamsGroup := e.Group(
		"/teams/:teamId",
		AuthGuard(),
		TeamGuard(),
	)

	teamsGroup.GET("", AuthHandler(h.GetTeam))
	teamsGroup.PUT("", AuthHandler(h.PutTeam), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	teamsGroup.GET("/users", AuthHandler(h.GetTeamUsers))
	teamsGroup.DELETE("/users/:userId", AuthHandler(h.DeleteTeamUser), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	teamsGroup.PUT("/users/:userId", AuthHandler(h.PutTeamUser), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	teamsGroup.PUT("/avatar", AuthHandler(h.PutTeamAvatar), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	teamsGroup.DELETE("/avatar", AuthHandler(h.DeleteTeamAvatar), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
}
