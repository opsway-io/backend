package changelogs

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
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
	h := &Handlers{
		TeamService: teamService,
	}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	changelogsGroup := e.Group(
		"/teams/:teamId/changelogs",
		TeamGuard(),
	)

	changelogsGroup.GET("", AuthHandler(h.GetChangelogs))
	changelogsGroup.POST("", AuthHandler(h.PostChangelogs), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	changelogsGroup.GET("/:changelogId", AuthHandler(h.GetChangelog))
	changelogsGroup.DELETE("/:changelogId", AuthHandler(h.DeleteChangelog), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	changelogsGroup.PUT("/:changelogId", AuthHandler(h.PutChangelog), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	changelogsGroup.GET("/:changelogId/entries", AuthHandler(h.GetChangelogEntries))
	changelogsGroup.POST("/:changelogId/entries", AuthHandler(h.PostChangelogEntries), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	changelogsGroup.GET("/:changelogId/entries/:entryId", AuthHandler(h.GetChangelogEntry))
	changelogsGroup.DELETE("/:changelogId/entries/:entryId", AuthHandler(h.DeleteChangelogEntry), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	changelogsGroup.PUT("/:changelogId/entries/:entryId", AuthHandler(h.PutChangelogEntry), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
}
