package incidents

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	TeamService     team.Service
	IncidentService incident.Service
	AlertingService alerting.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	incidentService incident.Service,
	alertingService alerting.Service,
) {
	h := &Handlers{
		TeamService:     teamService,
		IncidentService: incidentService,
		AlertingService: alertingService,
	}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	incidentsGroup := e.Group(
		"/teams/:teamId/incidents",
		TeamGuard(),
	)

	incidentsGroup.GET("", AuthHandler(h.GetIncidents))
	incidentsGroup.GET("/overview", AuthHandler(h.GetIncidents))
	incidentsGroup.GET("/monitor/:monitorId", AuthHandler(h.GetMonitorIncidents))
	incidentsGroup.GET("/:incidentId", AuthHandler(h.GetIncident))
	incidentsGroup.GET("/:incidentId/alerts", AuthHandler(h.GetIncidentAlerts))
	incidentsGroup.GET("/:incidentId/occurrences", AuthHandler(h.GetIncidentOccurrences))
	incidentsGroup.PATCH("/:incidentId/resolved", AuthHandler(h.PatchSolveMonitorIncident))
	incidentsGroup.PATCH("/:incidentId/acknowledge", AuthHandler(h.PatchAcknowledgeIncident))
	incidentsGroup.PATCH("/:incidentId/visibility", AuthHandler(h.PatchIncidentVisibility), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
}
