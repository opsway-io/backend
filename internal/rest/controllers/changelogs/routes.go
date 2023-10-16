package changelogs

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	TeamService           team.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
) {
	// h := &Handlers{
	// 	TeamService: teamService,
	// }

	// TeamGuard := mw.TeamGuardFactory(logger, teamService)
	// AllowedRoles := mw.RoleGuardFactory(logger, teamService)

	// AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	// changelogsGroup := e.Group(
	// 	"/teams/:teamId/changelogs",
	// 	TeamGuard(),
	// )

	// changelogsGroup.GET("", AuthHandler(h.TODO))
	// changelogsGroup.POST("", AuthHandler(h.TODO), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	// changelogsGroup.GET("/:changelogId", AuthHandler(h.TODO))
	// changelogsGroup.DELETE("/:changelogId", AuthHandler(h.TODO), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	// changelogsGroup.PUT("/:changelogId", AuthHandler(h.TODO), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	// changelogsGroup.GET("/:changelogId/entries", AuthHandler(h.TODO))
	// changelogsGroup.POST("/:changelogId/entries", AuthHandler(h.TODO), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	// changelogsGroup.GET("/:changelogId/entries/:entryId", AuthHandler(h.TODO))
	// changelogsGroup.DELETE("/:changelogId/entries/:entryId", AuthHandler(h.TODO), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	// changelogsGroup.PUT("/:changelogId/entries/:entryId", AuthHandler(h.TODO), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
}
