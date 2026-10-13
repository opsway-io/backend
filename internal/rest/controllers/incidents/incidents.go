package incidents

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/incident"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type IncidentUserResponse struct {
	ID          uint   `json:"id"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

func newIncidentUserResponse(u *entities.User) *IncidentUserResponse {
	if u == nil {
		return nil
	}

	displayName := u.Name
	if u.DisplayName != nil && *u.DisplayName != "" {
		displayName = *u.DisplayName
	}

	return &IncidentUserResponse{
		ID:          u.ID,
		DisplayName: displayName,
		Email:       u.Email,
	}
}

// IncidentResponse is the shared representation of an incident.
type IncidentResponse struct {
	ID                        uint                  `json:"id"`
	TeamID                    uint                  `json:"teamId"`
	MonitorID                 uint                  `json:"monitorId"`
	Title                     string                `json:"title"`
	Description               string                `json:"description"`
	Resolved                  bool                  `json:"resolved"`
	ResolvedAt                *time.Time            `json:"resolvedAt,omitempty"`
	ResolvedBy                *IncidentUserResponse `json:"resolvedBy,omitempty"`
	Acknowledged              bool                  `json:"acknowledged"`
	AcknowledgedAt            *time.Time            `json:"acknowledgedAt,omitempty"`
	AcknowledgedBy            *IncidentUserResponse `json:"acknowledgedBy,omitempty"`
	AcknowledgedByIntegration *string               `json:"acknowledgedByIntegration,omitempty"`
	RootCauseAnalysis         *string               `json:"rootCauseAnalysis,omitempty"`
	IsStatusPageVisible       bool                  `json:"isStatusPageVisible"`
	Occurrences               int64                 `json:"occurrences"`
	CreatedAt                 time.Time             `json:"createdAt"`
	UpdatedAt                 time.Time             `json:"updatedAt"`
}

func newIncidentResponse(in *entities.Incident, occurrences int64) IncidentResponse {
	description := ""
	if in.Description != nil {
		description = *in.Description
	}

	return IncidentResponse{
		ID:                        in.ID,
		TeamID:                    in.TeamID,
		MonitorID:                 in.MonitorID,
		Title:                     in.Title,
		Description:               description,
		Resolved:                  in.Resolved,
		ResolvedAt:                in.ResolvedAt,
		ResolvedBy:                newIncidentUserResponse(in.ResolvedBy),
		Acknowledged:              in.Acknowledged,
		AcknowledgedAt:            in.AcknowledgedAt,
		AcknowledgedBy:            newIncidentUserResponse(in.AcknowledgedBy),
		AcknowledgedByIntegration: in.AcknowledgedByIntegration,
		RootCauseAnalysis:         in.RootCauseAnalysis,
		IsStatusPageVisible:       in.IsStatusPageVisible,
		Occurrences:               occurrences,
		CreatedAt:                 in.CreatedAt,
		UpdatedAt:                 in.UpdatedAt,
	}
}

func incidentError(c hs.AuthenticatedContext, err error, msg string) error {
	if errors.Is(err, incident.ErrNotFound) {
		return echo.ErrNotFound
	}

	if errors.Is(err, incident.ErrUnresolvedIncidentExists) {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	c.Log.WithError(err).Error(msg)

	return echo.ErrInternalServerError
}

/*
	List
*/

type GetIncidentsRequest struct {
	TeamID   uint  `param:"teamId" validate:"required,numeric,gte=0"`
	Resolved *bool `query:"resolved"`
	Offset   *int  `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit    *int  `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetIncidentsResponse struct {
	Incidents []IncidentResponse `json:"incidents"`
}

func (h *Handlers) GetIncidents(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentsRequest")

		return echo.ErrBadRequest
	}

	incidents, err := h.IncidentService.GetByTeamIDPaginated(
		c.Request().Context(),
		req.TeamID,
		req.Resolved,
		req.Offset,
		req.Limit,
	)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incidents")

		return echo.ErrInternalServerError
	}

	resp := GetIncidentsResponse{
		Incidents: make([]IncidentResponse, len(*incidents)),
	}
	for i, in := range *incidents {
		resp.Incidents[i] = newIncidentResponse(&in.Incident, in.OccurrenceCount)
	}

	return c.JSON(http.StatusOK, resp)
}

/*
	Get by ID
*/

type GetIncidentRequest struct {
	TeamID     uint `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint `param:"incidentId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetIncident(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	in, err := h.IncidentService.GetByTeamIDAndID(ctx, req.TeamID, req.IncidentID)
	if err != nil {
		return incidentError(c, err, "failed to get incident")
	}

	occurrences, err := h.IncidentService.CountOccurrences(ctx, in.ID)
	if err != nil {
		return incidentError(c, err, "failed to count incident occurrences")
	}

	return c.JSON(http.StatusOK, newIncidentResponse(in, occurrences))
}

/*
	Monitor incidents
*/

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
	IncidentResponse
	Property string `json:"property"`
	Target   string `json:"target"`
	Operator string `json:"operator"`
}

func (h *Handlers) GetMonitorIncidents(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorIncidentsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorIncidentsRequest")

		return echo.ErrBadRequest
	}

	incidents, err := h.IncidentService.GetByMonitorIDWithAssertionPaginated(c.Request().Context(), req.MonitorID, req.Offset, req.Limit)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incidents")

		return echo.ErrInternalServerError
	}

	resp := GetMonitorIncidentsResponse{
		Incidents: []GetMonitorIncidentsResponseIncident{},
	}
	for _, in := range *incidents {
		// Guard against monitors of other teams
		if in.TeamID != req.TeamID {
			continue
		}

		resp.Incidents = append(resp.Incidents, GetMonitorIncidentsResponseIncident{
			IncidentResponse: newIncidentResponse(&in.Incident, 0),
			Property:         in.Property,
			Target:           in.Target,
			Operator:         in.Operator,
		})
	}

	return c.JSON(http.StatusOK, resp)
}

/*
	Mutations
*/

type PatchSolveMonitorIncidentRequest struct {
	TeamID     uint  `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint  `param:"incidentId" validate:"required,numeric,gte=0"`
	Resolved   *bool `json:"resolved" validate:"required"`
}

func (h *Handlers) PatchSolveMonitorIncident(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PatchSolveMonitorIncidentRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PatchSolveMonitorIncidentRequest")

		return echo.ErrBadRequest
	}

	if err := h.IncidentService.SetResolved(c.Request().Context(), req.TeamID, req.IncidentID, *req.Resolved, c.UserID); err != nil {
		return incidentError(c, err, "failed to update incident")
	}

	return c.NoContent(http.StatusNoContent)
}

type PatchAcknowledgeIncidentRequest struct {
	TeamID     uint `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint `param:"incidentId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) PatchAcknowledgeIncident(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PatchAcknowledgeIncidentRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PatchAcknowledgeIncidentRequest")

		return echo.ErrBadRequest
	}

	if err := h.IncidentService.Acknowledge(c.Request().Context(), req.TeamID, req.IncidentID, c.UserID); err != nil {
		return incidentError(c, err, "failed to acknowledge incident")
	}

	return c.NoContent(http.StatusNoContent)
}

type PatchIncidentVisibilityRequest struct {
	TeamID              uint  `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID          uint  `param:"incidentId" validate:"required,numeric,gte=0"`
	IsStatusPageVisible *bool `json:"isStatusPageVisible" validate:"required"`
}

func (h *Handlers) PatchIncidentVisibility(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PatchIncidentVisibilityRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PatchIncidentVisibilityRequest")

		return echo.ErrBadRequest
	}

	if err := h.IncidentService.SetStatusPageVisibility(c.Request().Context(), req.TeamID, req.IncidentID, *req.IsStatusPageVisible); err != nil {
		return incidentError(c, err, "failed to update incident visibility")
	}

	return c.NoContent(http.StatusNoContent)
}

/*
	Alerts & occurrences
*/

type GetIncidentSubResourceRequest struct {
	TeamID     uint `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint `param:"incidentId" validate:"required,numeric,gte=0"`
	Offset     *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit      *int `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetIncidentAlertsResponse struct {
	TotalCount int64                            `json:"totalCount"`
	Alerts     []GetIncidentAlertsResponseAlert `json:"alerts"`
}

type GetIncidentAlertsResponseAlert struct {
	ID          uint      `json:"id"`
	AlertRuleID uint      `json:"alertRuleId"`
	Channels    string    `json:"channels"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (h *Handlers) GetIncidentAlerts(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentSubResourceRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentSubResourceRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	if _, err := h.IncidentService.GetByTeamIDAndID(ctx, req.TeamID, req.IncidentID); err != nil {
		return incidentError(c, err, "failed to get incident")
	}

	triggers, total, err := h.AlertingService.GetTriggersByIncidentID(ctx, req.TeamID, req.IncidentID, req.Offset, req.Limit)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident alerts")

		return echo.ErrInternalServerError
	}

	resp := GetIncidentAlertsResponse{
		TotalCount: total,
		Alerts:     make([]GetIncidentAlertsResponseAlert, len(*triggers)),
	}
	for i, t := range *triggers {
		resp.Alerts[i] = GetIncidentAlertsResponseAlert{
			ID:          t.ID,
			AlertRuleID: t.AlertRuleID,
			Channels:    t.Channels,
			CreatedAt:   t.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, resp)
}

type GetIncidentOccurrencesResponse struct {
	TotalCount  int64                                      `json:"totalCount"`
	Occurrences []GetIncidentOccurrencesResponseOccurrence `json:"occurrences"`
}

type GetIncidentOccurrencesResponseOccurrence struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *Handlers) GetIncidentOccurrences(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentSubResourceRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentSubResourceRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	if _, err := h.IncidentService.GetByTeamIDAndID(ctx, req.TeamID, req.IncidentID); err != nil {
		return incidentError(c, err, "failed to get incident")
	}

	occurrences, total, err := h.IncidentService.GetOccurrencesPaginated(ctx, req.IncidentID, req.Offset, req.Limit)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident occurrences")

		return echo.ErrInternalServerError
	}

	resp := GetIncidentOccurrencesResponse{
		TotalCount:  total,
		Occurrences: make([]GetIncidentOccurrencesResponseOccurrence, len(*occurrences)),
	}
	for i, o := range *occurrences {
		resp.Occurrences[i] = GetIncidentOccurrencesResponseOccurrence{
			ID:        o.ID,
			CreatedAt: o.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, resp)
}
