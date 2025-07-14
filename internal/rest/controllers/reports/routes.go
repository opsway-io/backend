package reports

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/check"
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
	CheckService          check.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	reportService report.Service,
	checkService check.Service,
) {
	h := &Handlers{
		ReportService: reportService,
		CheckService:  checkService,
	}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	reportsGroup := e.Group(
		"/teams/:teamId/reports",
		TeamGuard(),
	)

	reportsGroup.GET("", AuthHandler(h.GetReports))
	reportsGroup.POST("", AuthHandler(h.CreateReport))
}
