package reports

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/report"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	TeamService           team.Service
	ReportService         report.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	reportService report.Service,
) {
	h := &Handlers{
		ReportService: reportService,
	}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	monitorsGroup := e.Group(
		"/teams/:teamId/reports",
		TeamGuard(),
	)

	monitorsGroup.GET("", AuthHandler(h.GetReports))
}
