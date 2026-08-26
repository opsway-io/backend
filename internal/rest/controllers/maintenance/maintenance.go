package maintenance

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/maintenance"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	TeamService        team.Service
	MaintenanceService maintenance.Service
	MonitorService     monitor.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	maintenanceService maintenance.Service,
	monitorService monitor.Service,
) {
	h := &Handlers{
		TeamService:        teamService,
		MaintenanceService: maintenanceService,
		MonitorService:     monitorService,
	}

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	cg := e.Group("/teams/:teamId/maintenance")

	cg.GET("", AuthHandler(h.GetMaintenanceWindows))
	cg.POST("", AuthHandler(h.PostMaintenanceWindow))
	cg.GET("/:maintenanceId", AuthHandler(h.GetMaintenanceWindow))
	cg.PUT("/:maintenanceId", AuthHandler(h.PutMaintenanceWindow))
	cg.DELETE("/:maintenanceId", AuthHandler(h.DeleteMaintenanceWindow))
}

type GetMaintenanceWindowsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetMaintenanceWindows(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMaintenanceWindowsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMaintenanceWindowsRequest")
		return echo.ErrBadRequest
	}

	maintenances, err := h.MaintenanceService.GetByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get maintenance windows")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, maintenances)
}

type GetMaintenanceWindowRequest struct {
	TeamID        uint `param:"teamId" validate:"required,numeric,gte=0"`
	MaintenanceID uint `param:"maintenanceId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetMaintenanceWindow(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMaintenanceWindowRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMaintenanceWindowRequest")
		return echo.ErrBadRequest
	}

	m, err := h.MaintenanceService.GetByID(c.Request().Context(), req.MaintenanceID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get maintenance window")
		return echo.ErrNotFound
	}

	return c.JSON(http.StatusOK, m)
}

type PostMaintenanceWindowRequest struct {
	TeamID      uint      `param:"teamId" validate:"required,numeric,gte=0"`
	Title       string    `json:"title" validate:"required"`
	Description *string   `json:"description"`
	StartAt     time.Time `json:"startAt" validate:"required"`
	EndAt       time.Time `json:"endAt" validate:"required,gtfield=StartAt"`
	MonitorIDs  []uint    `json:"monitorIds"`
}

func (h *Handlers) PostMaintenanceWindow(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PostMaintenanceWindowRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostMaintenanceWindowRequest")
		fmt.Println("BIND ERROR POST MAINTENANCE:", err)
		return echo.ErrBadRequest
	}

	var monitors []entities.Monitor
	if req.MonitorIDs != nil {
		monitors = make([]entities.Monitor, 0, len(req.MonitorIDs))
		for _, id := range req.MonitorIDs {
			m, err := h.MonitorService.GetMonitorAndSettingsByTeamIDAndID(c.Request().Context(), req.TeamID, id)
			if err != nil {
				c.Log.WithError(err).Debug("failed to get monitor")
				return echo.ErrBadRequest
			}
			monitors = append(monitors, *m)
		}
	}

	m := &entities.Maintenance{
		TeamID:      req.TeamID,
		Title:       req.Title,
		Description: req.Description,
		Settings: entities.MaintenanceSettings{
			StartAt: req.StartAt,
			EndAt:   req.EndAt,
		},
		Monitors: monitors,
	}

	err = h.MaintenanceService.Create(c.Request().Context(), m)
	if err != nil {
		c.Log.WithError(err).Error("failed to create maintenance window")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, m)
}

type PutMaintenanceWindowRequest struct {
	TeamID        uint      `param:"teamId" validate:"required,numeric,gte=0"`
	MaintenanceID uint      `param:"maintenanceId" validate:"required,numeric,gte=0"`
	Title         string    `json:"title" validate:"required"`
	Description   *string   `json:"description"`
	StartAt       time.Time `json:"startAt" validate:"required"`
	EndAt         time.Time `json:"endAt" validate:"required,gtfield=StartAt"`
	MonitorIDs    []uint    `json:"monitorIds"`
}

func (h *Handlers) PutMaintenanceWindow(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PutMaintenanceWindowRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutMaintenanceWindowRequest")
		return echo.ErrBadRequest
	}

	m, err := h.MaintenanceService.GetByID(c.Request().Context(), req.MaintenanceID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get maintenance window for update")
		return echo.ErrNotFound
	}

	var monitors []entities.Monitor
	if req.MonitorIDs != nil {
		monitors = make([]entities.Monitor, 0, len(req.MonitorIDs))
		for _, id := range req.MonitorIDs {
			monitorEntity, err := h.MonitorService.GetMonitorAndSettingsByTeamIDAndID(c.Request().Context(), req.TeamID, id)
			if err != nil {
				c.Log.WithError(err).Debug("failed to get monitor")
				return echo.ErrBadRequest
			}
			monitors = append(monitors, *monitorEntity)
		}
	}

	m.Title = req.Title
	m.Description = req.Description
	m.Settings.StartAt = req.StartAt
	m.Settings.EndAt = req.EndAt
	m.Monitors = monitors

	err = h.MaintenanceService.Update(c.Request().Context(), m)
	if err != nil {
		c.Log.WithError(err).Error("failed to update maintenance window")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, m)
}

type DeleteMaintenanceWindowRequest struct {
	TeamID        uint `param:"teamId" validate:"required,numeric,gte=0"`
	MaintenanceID uint `param:"maintenanceId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteMaintenanceWindow(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteMaintenanceWindowRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteMaintenanceWindowRequest")
		return echo.ErrBadRequest
	}

	m, err := h.MaintenanceService.GetByID(c.Request().Context(), req.MaintenanceID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get maintenance window for deletion")
		return echo.ErrNotFound
	}

	err = h.MaintenanceService.Delete(c.Request().Context(), m)
	if err != nil {
		c.Log.WithError(err).Error("failed to delete maintenance window")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
