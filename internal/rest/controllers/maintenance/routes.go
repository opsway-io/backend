package maintenance

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/maintenance"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	MaintenanceService maintenance.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	maintenanceService maintenance.Service,
) {
	h := &Handlers{
		MaintenanceService: maintenanceService,
	}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)
	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	group := e.Group(
		"/teams/:teamId/maintenance",
		TeamGuard(),
	)

	group.GET("", AuthHandler(h.GetMaintenanceWindows))
	group.POST("", AuthHandler(h.PostMaintenanceWindow), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	group.GET("/:maintenanceId", AuthHandler(h.GetMaintenanceWindow))
	group.PUT("/:maintenanceId", AuthHandler(h.PutMaintenanceWindow), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	group.DELETE("/:maintenanceId", AuthHandler(h.DeleteMaintenanceWindow), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
}
