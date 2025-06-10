package incidents

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	TeamService           team.Service
	IncidentService       incident.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	incidentService incident.Service,
) {
	h := &Handlers{
		IncidentService: incidentService,
	}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	monitorsGroup := e.Group(
		"/teams/:teamId/incidents",
		TeamGuard(),
	)

	monitorsGroup.GET("", AuthHandler(h.GetIncidents))
	monitorsGroup.GET("/overview", AuthHandler(h.GetIncidents))
	monitorsGroup.GET("/monitor/:monitorId", AuthHandler(h.GetIncidents))
}
