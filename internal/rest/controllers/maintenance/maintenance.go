package maintenance

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/maintenance"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type MaintenanceWindowResponse struct {
	ID          uint                               `json:"id"`
	Title       string                             `json:"title"`
	Description *string                            `json:"description"`
	Status      entities.MaintenanceStatus         `json:"status"`
	Settings    MaintenanceWindowResponseSettings  `json:"settings"`
	Monitors    []MaintenanceWindowResponseMonitor `json:"monitors"`
	CreatedAt   time.Time                          `json:"createdAt"`
	UpdatedAt   time.Time                          `json:"updatedAt"`
}

type MaintenanceWindowResponseSettings struct {
	StartAt time.Time `json:"startAt"`
	EndAt   time.Time `json:"endAt"`
}

type MaintenanceWindowResponseMonitor struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

func newMaintenanceWindowResponse(m *entities.Maintenance, now time.Time) MaintenanceWindowResponse {
	monitors := make([]MaintenanceWindowResponseMonitor, len(m.Monitors))
	for i, mon := range m.Monitors {
		monitors[i] = MaintenanceWindowResponseMonitor{ID: mon.ID, Name: mon.Name}
	}

	return MaintenanceWindowResponse{
		ID:          m.ID,
		Title:       m.Title,
		Description: m.Description,
		Status:      m.StatusAt(now),
		Settings: MaintenanceWindowResponseSettings{
			StartAt: m.Settings.StartAt,
			EndAt:   m.Settings.EndAt,
		},
		Monitors:  monitors,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type GetMaintenanceWindowsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetMaintenanceWindows(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMaintenanceWindowsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMaintenanceWindowsRequest")

		return echo.ErrBadRequest
	}

	windows, err := h.MaintenanceService.GetByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get maintenance windows")

		return echo.ErrInternalServerError
	}

	now := time.Now()
	resp := make([]MaintenanceWindowResponse, len(*windows))
	for i := range *windows {
		resp[i] = newMaintenanceWindowResponse(&(*windows)[i], now)
	}

	return c.JSON(http.StatusOK, resp)
}

type GetMaintenanceWindowRequest struct {
	TeamID        uint `param:"teamId" validate:"required,numeric,gte=0"`
	MaintenanceID uint `param:"maintenanceId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetMaintenanceWindow(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMaintenanceWindowRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMaintenanceWindowRequest")

		return echo.ErrBadRequest
	}

	window, err := h.MaintenanceService.GetByTeamIDAndID(c.Request().Context(), req.TeamID, req.MaintenanceID)
	if err != nil {
		if errors.Is(err, maintenance.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to get maintenance window")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, newMaintenanceWindowResponse(window, time.Now()))
}

type PostMaintenanceWindowRequest struct {
	TeamID      uint      `param:"teamId" validate:"required,numeric,gte=0"`
	Title       string    `json:"title" validate:"required,min=1,max=255"`
	Description *string   `json:"description" validate:"omitempty,max=10000"`
	StartAt     time.Time `json:"startAt" validate:"required"`
	EndAt       time.Time `json:"endAt" validate:"required,gtfield=StartAt"`
	MonitorIDs  []uint    `json:"monitorIds" validate:"omitempty,max=1000,dive,gt=0"`
}

func (h *Handlers) PostMaintenanceWindow(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostMaintenanceWindowRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostMaintenanceWindowRequest")

		return echo.ErrBadRequest
	}

	window := &entities.Maintenance{
		TeamID:      req.TeamID,
		Title:       strings.TrimSpace(req.Title),
		Description: normalizeDescription(req.Description),
		Settings: entities.MaintenanceSettings{
			StartAt: req.StartAt.UTC(),
			EndAt:   req.EndAt.UTC(),
		},
	}

	if err := h.MaintenanceService.Create(c.Request().Context(), window, req.MonitorIDs); err != nil {
		return h.mutationError(c, err)
	}

	return c.JSON(http.StatusCreated, newMaintenanceWindowResponse(window, time.Now()))
}

type PutMaintenanceWindowRequest struct {
	TeamID        uint      `param:"teamId" validate:"required,numeric,gte=0"`
	MaintenanceID uint      `param:"maintenanceId" validate:"required,numeric,gte=0"`
	Title         string    `json:"title" validate:"required,min=1,max=255"`
	Description   *string   `json:"description" validate:"omitempty,max=10000"`
	StartAt       time.Time `json:"startAt" validate:"required"`
	EndAt         time.Time `json:"endAt" validate:"required,gtfield=StartAt"`
	MonitorIDs    []uint    `json:"monitorIds" validate:"omitempty,max=1000,dive,gt=0"`
}

// PutMaintenanceWindow replaces the window. Completing a window early is done by
// setting endAt to the current time.
func (h *Handlers) PutMaintenanceWindow(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutMaintenanceWindowRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutMaintenanceWindowRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	window := &entities.Maintenance{
		ID:          req.MaintenanceID,
		TeamID:      req.TeamID,
		Title:       strings.TrimSpace(req.Title),
		Description: normalizeDescription(req.Description),
		Settings: entities.MaintenanceSettings{
			StartAt: req.StartAt.UTC(),
			EndAt:   req.EndAt.UTC(),
		},
	}

	if err := h.MaintenanceService.Update(ctx, window, req.MonitorIDs); err != nil {
		return h.mutationError(c, err)
	}

	updated, err := h.MaintenanceService.GetByTeamIDAndID(ctx, req.TeamID, req.MaintenanceID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get maintenance window")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, newMaintenanceWindowResponse(updated, time.Now()))
}

type DeleteMaintenanceWindowRequest struct {
	TeamID        uint `param:"teamId" validate:"required,numeric,gte=0"`
	MaintenanceID uint `param:"maintenanceId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteMaintenanceWindow(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteMaintenanceWindowRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteMaintenanceWindowRequest")

		return echo.ErrBadRequest
	}

	if err := h.MaintenanceService.Delete(c.Request().Context(), req.TeamID, req.MaintenanceID); err != nil {
		return h.mutationError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handlers) mutationError(c hs.AuthenticatedContext, err error) error {
	switch {
	case errors.Is(err, maintenance.ErrNotFound):
		return echo.ErrNotFound
	case errors.Is(err, maintenance.ErrInvalidMonitor):
		return echo.NewHTTPError(http.StatusBadRequest, "one or more monitors do not exist")
	default:
		c.Log.WithError(err).Error("failed to mutate maintenance window")

		return echo.ErrInternalServerError
	}
}

func normalizeDescription(d *string) *string {
	if d == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*d)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
