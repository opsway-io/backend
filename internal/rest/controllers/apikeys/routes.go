package apikeys

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/apikey"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	ApiKeyService apikey.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	apiKeyService apikey.Service,
) {
	h := &Handlers{ApiKeyService: apiKeyService}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)
	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	g := e.Group("/teams/:teamId/apikeys", TeamGuard(), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	g.GET("", AuthHandler(h.GetApiKeys))
	g.POST("", AuthHandler(h.PostApiKey))
	g.DELETE("/:keyId", AuthHandler(h.DeleteApiKey))
}
