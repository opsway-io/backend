package incidents

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/incident"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetIncidentsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
	Offset *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit  *int `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetIncidentsResponse struct {
	Incidents []GetIncidentsResponseIncident `json:"incidents"`
}

type GetIncidentsResponseIncident struct {
	ID          uint   `json:"id"`
	TeamID      uint   `json:"teamId"`
	MonitorID   uint   `json:"monitorId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
}

func (h *Handlers) GetIncidents(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	incidents, err := h.IncidentService.GetByTeamIDPaginated(
		ctx,
		req.TeamID,
		req.Offset,
		req.Limit)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incidents")

		return echo.ErrInternalServerError
	}

	resp := h.newGetIncidentResponse(incidents)

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) newGetIncidentResponse(incidents *[]entities.Incident) *GetIncidentsResponse {
	resp := &GetIncidentsResponse{
		Incidents: make([]GetIncidentsResponseIncident, len(*incidents)),
	}

	for i, in := range *incidents {
		resp.Incidents[i] = GetIncidentsResponseIncident{
			ID:          in.ID,
			TeamID:      in.TeamID,
			MonitorID:   in.MonitorID,
			Title:       in.Title,
			Description: *in.Description,
			CreatedAt:   in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return resp
}

type GetIncidentOverviewRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
	Offset *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit  *int `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetIncidentOverviewResponse struct {
	Checks []GetIncidentOverviewResponseIncident `json:"incidents"`
}

type GetIncidentOverviewResponseIncident struct {
	ID        uint   `json:"id"`
	TeamID    uint   `json:"teamId"`
	MonitorID uint   `json:"monitorId"`
	CreatedAt string `json:"createdAt"`
	Count     int    `json:"count"`
}

func (h *Handlers) GetIncidentOverview(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentOverviewRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	incidents, err := h.IncidentService.GetByTeamIDPaginated(
		ctx,
		req.TeamID,
		req.Offset,
		req.Limit)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incidents")

		return echo.ErrInternalServerError
	}

	resp := h.newGetIncidentOverviewResponse(incidents)

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) newGetIncidentOverviewResponse(incidents *[]entities.Incident) *GetIncidentOverviewResponse {

	resp := &GetIncidentOverviewResponse{
		Checks: make([]GetIncidentOverviewResponseIncident, len(*incidents)),
	}

	for i, incident := range *incidents {
		resp.Checks[i] = GetIncidentOverviewResponseIncident{
			ID:        incident.ID,
			TeamID:    incident.TeamID,
			MonitorID: incident.MonitorID,
			Count:     0,
			CreatedAt: incident.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return resp
}

type GetMonitorIncidentsRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
	Offset    *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit     *int `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetMonitorIncidentsResponse struct {
	Incidents []GetMonitorIncidentsResponseIncident `json:"incidents"`
}

type GetMonitorIncidentsResponseIncident struct {
	ID          uint   `json:"id"`
	TeamID      uint   `json:"teamId"`
	MonitorID   uint   `json:"monitorId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Property    string `json:"property"`
	Target      string `json:"target"`
	Operator    string `json:"operator"`
}

func (h *Handlers) GetMonitorIncidents(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorIncidentsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorIncidentsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	incidents, err := h.IncidentService.GetByMonitorIDWithAssertionPaginated(
		ctx,
		req.MonitorID,
		req.Offset,
		req.Limit)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incidents")

		return echo.ErrInternalServerError
	}

	resp := h.GetMonitorIncidentsResponse(incidents)

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) GetMonitorIncidentsResponse(incidents *[]incident.IncidentAndAssertion) *GetMonitorIncidentsResponse {
	resp := &GetMonitorIncidentsResponse{
		Incidents: make([]GetMonitorIncidentsResponseIncident, len(*incidents)),
	}

	for i, in := range *incidents {
		resp.Incidents[i] = GetMonitorIncidentsResponseIncident{
			ID:          in.ID,
			TeamID:      in.TeamID,
			MonitorID:   in.MonitorID,
			Title:       in.Title,
			Description: *in.Description,
			CreatedAt:   in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   in.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Property:    in.Property,
			Target:      in.Target,
			Operator:    in.Operator,
		}
	}

	return resp
}

type PatchSolveMonitorIncidentRequset struct {
	TeamID     uint `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint `param:"incidentId" validate:"required,numeric,gte=0"`
	Resolved   bool `json:"resolved" validate:"required"`
}

func (h *Handlers) PatchSolveMonitorIncident(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PatchSolveMonitorIncidentRequset](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PatchSolveMonitorIncidentRequset")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	incident, err := h.IncidentService.GetByID(ctx, req.IncidentID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident")

		return echo.ErrInternalServerError
	}

	incident.Resolved = req.Resolved

	err = h.IncidentService.Update(
		ctx,
		incident,
	)
	if err != nil {
		c.Log.WithError(err).Error("failed to update incident")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
