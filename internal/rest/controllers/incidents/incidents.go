package incidents

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/opsway-io/backend/internal/incident"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetIncidentsRequest struct {
	TeamID   uint  `param:"teamId" validate:"required,numeric,gte=0"`
	Resolved *bool `query:"resolved" validate:"omitempty"`
	Offset   *int  `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit    *int  `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetIncidentsResponse struct {
	Incidents []GetIncidentsResponseIncident `json:"incidents"`
}

type GetIncidentsResponseIncident struct {
	ID                uint    `json:"id"`
	TeamID            uint    `json:"teamId"`
	MonitorID         *uint   `json:"monitorId"`
	HeartbeatID       *uint   `json:"heartbeatId"`
	Title             string  `json:"title"`
	Description       string  `json:"description"`
	RootCauseAnalysis *string `json:"rootCauseAnalysis,omitempty"`
	Resolved          bool    `json:"resolved"`
	CreatedAt         string  `json:"createdAt"`
}

func (h *Handlers) GetIncidents(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	incidents, err := h.IncidentService.GetByTeamIDAndStatusPaginated(
		ctx,
		req.TeamID,
		req.Resolved,
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
			ID:                in.ID,
			TeamID:            in.TeamID,
			MonitorID:         in.MonitorID,
			HeartbeatID:       in.HeartbeatID,
			Title:             in.Title,
			Description:       *in.Description,
			RootCauseAnalysis: in.RootCauseAnalysis,
			Resolved:          in.Resolved,
			CreatedAt:         in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
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
	ID          uint   `json:"id"`
	TeamID      uint   `json:"teamId"`
	MonitorID   *uint  `json:"monitorId"`
	HeartbeatID *uint  `json:"heartbeatId"`
	CreatedAt   string `json:"createdAt"`
	Count       int    `json:"count"`
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
			ID:          incident.ID,
			TeamID:      incident.TeamID,
			MonitorID:   incident.MonitorID,
			HeartbeatID: incident.HeartbeatID,
			Count:       0,
			CreatedAt:   incident.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
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
	ID                uint    `json:"id"`
	TeamID            uint    `json:"teamId"`
	MonitorID         *uint   `json:"monitorId"`
	HeartbeatID       *uint   `json:"heartbeatId"`
	Title             string  `json:"title"`
	Description       string  `json:"description"`
	RootCauseAnalysis *string `json:"rootCauseAnalysis,omitempty"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
	Property          string  `json:"property"`
	Target            string  `json:"target"`
	Operator          string  `json:"operator"`
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
		property := ""
		if in.Property != nil {
			property = *in.Property
		}

		target := ""
		if in.Target != nil {
			target = *in.Target
		}

		operator := ""
		if in.Operator != nil {
			operator = *in.Operator
		}

		resp.Incidents[i] = GetMonitorIncidentsResponseIncident{
			ID:                in.ID,
			TeamID:            in.TeamID,
			MonitorID:         in.MonitorID,
			HeartbeatID:       in.HeartbeatID,
			Title:             in.Title,
			Description:       *in.Description,
			RootCauseAnalysis: in.RootCauseAnalysis,
			CreatedAt:         in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:         in.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Property:          property,
			Target:            target,
			Operator:          operator,
		}
	}

	return resp
}

type GetIncidentRequest struct {
	TeamID     uint `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint `param:"incidentId" validate:"required,numeric,gte=0"`
}

type GetIncidentResponse struct {
	ID                uint    `json:"id"`
	TeamID            uint    `json:"teamId"`
	MonitorID         *uint   `json:"monitorId"`
	HeartbeatID       *uint   `json:"heartbeatId"`
	Title             string  `json:"title"`
	Description       string  `json:"description"`
	Resolved          bool    `json:"resolved"`
	Acknowledged      bool    `json:"acknowledged"`
	AcknowledgedAt    *string `json:"acknowledgedAt,omitempty"`
	RootCauseAnalysis *string `json:"rootCauseAnalysis,omitempty"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
	Occurrences       int     `json:"occurrences"`
}

func (h *Handlers) GetIncident(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()
	in, err := h.IncidentService.GetByID(ctx, req.IncidentID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident")
		return echo.ErrInternalServerError
	}

	// Verify it belongs to the team
	if in.TeamID != req.TeamID {
		return echo.ErrForbidden
	}

	resp := &GetIncidentResponse{
		ID:           in.ID,
		TeamID:       in.TeamID,
		MonitorID:    in.MonitorID,
		HeartbeatID:  in.HeartbeatID,
		Title:        in.Title,
		Description:  *in.Description,
		Resolved:     in.Resolved,
		Acknowledged: in.Acknowledged,
		Occurrences:  in.Occurrences,
	}

	if in.AcknowledgedAt != nil {
		ackAt := in.AcknowledgedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.AcknowledgedAt = &ackAt
	}

	if in.RootCauseAnalysis != nil {
		resp.RootCauseAnalysis = in.RootCauseAnalysis
	}

	resp.CreatedAt = in.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	resp.UpdatedAt = in.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")

	return c.JSON(http.StatusOK, resp)
}

type PatchSolveIncidentRequest struct {
	TeamID     uint `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint `param:"incidentId" validate:"required,numeric,gte=0"`
	Resolved   bool `json:"resolved" validate:"required"`
}

func (h *Handlers) PatchSolveIncident(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PatchSolveIncidentRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PatchSolveIncidentRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()
	in, err := h.IncidentService.GetByID(ctx, req.IncidentID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident")
		return echo.ErrInternalServerError
	}

	// Verify it belongs to the team
	if in.TeamID != req.TeamID {
		return echo.ErrForbidden
	}

	in.Resolved = req.Resolved
	if err := h.IncidentService.Update(ctx, in); err != nil {
		c.Log.WithError(err).Error("failed to update incident")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handlers) PatchAcknowledgeIncident(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()
	in, err := h.IncidentService.GetByID(ctx, req.IncidentID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident")
		return echo.ErrInternalServerError
	}

	if in.TeamID != req.TeamID {
		return echo.ErrForbidden
	}

	in.Acknowledged = true
	now := time.Now()
	in.AcknowledgedAt = &now
	if err := h.IncidentService.Update(ctx, in); err != nil {
		c.Log.WithError(err).Error("failed to update incident")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusOK)
}

type PatchRootCauseIncidentRequest struct {
	TeamID            uint    `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID        uint    `param:"incidentId" validate:"required,numeric,gte=0"`
	RootCauseAnalysis *string `json:"rootCauseAnalysis"`
}

func (h *Handlers) PatchRootCauseIncident(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PatchRootCauseIncidentRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PatchRootCauseIncidentRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()
	in, err := h.IncidentService.GetByID(ctx, req.IncidentID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident")
		return echo.ErrInternalServerError
	}

	if in.TeamID != req.TeamID {
		return echo.ErrForbidden
	}

	in.RootCauseAnalysis = req.RootCauseAnalysis
	if req.RootCauseAnalysis != nil && *req.RootCauseAnalysis != "" && !in.RootCauseNotified {
		in.RootCauseNotified = true
		_ = h.EventService.Publish(events.IncidentPostMortemPublishedEvent{
			Incident: in,
		})
	}

	if err := h.IncidentService.Update(ctx, in); err != nil {
		c.Log.WithError(err).Error("failed to update incident")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusOK)
}

type GetIncidentOccurrencesRequest struct {
	TeamID     uint `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint `param:"incidentId" validate:"required,numeric,gte=0"`
	Offset     *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit      *int `query:"limit" validate:"omitempty,numeric,gt=0"`
}

type GetIncidentOccurrencesResponse struct {
	Occurrences []IncidentOccurrenceResponse `json:"occurrences"`
}

type IncidentOccurrenceResponse struct {
	ID        uint   `json:"id"`
	CreatedAt string `json:"createdAt"`
}

func (h *Handlers) GetIncidentOccurrences(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentOccurrencesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentOccurrencesRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	// Verify incident belongs to team
	in, err := h.IncidentService.GetByID(ctx, req.IncidentID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident")
		return echo.ErrInternalServerError
	}
	if in.TeamID != req.TeamID {
		return echo.ErrForbidden
	}

	occurrences, err := h.IncidentService.GetOccurrencesPaginated(ctx, req.IncidentID, req.Offset, req.Limit)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident occurrences")
		return echo.ErrInternalServerError
	}

	resp := GetIncidentOccurrencesResponse{
		Occurrences: make([]IncidentOccurrenceResponse, len(*occurrences)),
	}

	for i, occ := range *occurrences {
		resp.Occurrences[i] = IncidentOccurrenceResponse{
			ID:        occ.ID,
			CreatedAt: occ.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, resp)
}
