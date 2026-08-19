package reports

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event/events"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetReportsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

type GetReportsResponse struct {
	Reports []GetReportsResponseReport `json:"reports"`
}

type GetReportsResponseReport struct {
	ID        uint   `json:"id"`
	TeamID    uint   `json:"teamId"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type GetReportRequest struct {
	TeamID   uint `param:"teamId" validate:"required,numeric,gte=0"`
	ReportID uint `param:"reportId" validate:"required,numeric,gte=0"`
}

type GetReportResponse struct {
	ID        uint                `json:"id"`
	TeamID    uint                `json:"teamId"`
	Type      string              `json:"type"`
	Status    string              `json:"status"`
	CreatedAt string              `json:"createdAt"`
	Data      entities.ReportData `json:"data"`
}

func (h *Handlers) GetReports(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetReportsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetReportsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	reports, err := h.ReportService.GetResportsByTeam(
		ctx,
		req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get Reports")

		return echo.ErrInternalServerError
	}

	resp := h.newGetReportResponse(reports)

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) GetReport(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetReportRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetReportRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	report, err := h.ReportService.GetByID(ctx, req.ReportID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get Report")
		return echo.ErrNotFound
	}

	if report.TeamID != req.TeamID {
		return echo.ErrForbidden
	}

	resp := GetReportResponse{
		ID:        report.ID,
		TeamID:    report.TeamID,
		Type:      string(report.Type),
		Status:    string(report.Status),
		CreatedAt: report.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Data:      report.Report.Data(),
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) newGetReportResponse(reports *[]entities.Report) *GetReportsResponse {
	resp := &GetReportsResponse{
		Reports: make([]GetReportsResponseReport, len(*reports)),
	}

	for i, in := range *reports {
		resp.Reports[i] = GetReportsResponseReport{
			ID:        in.ID,
			TeamID:    in.TeamID,
			Type:      string(in.Type),
			Status:    string(in.Status),
			CreatedAt: in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return resp
}

type PostReportsRequest struct {
	TeamID     uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ReportType string `json:"reportType" validate:"required,oneof=UPTIME PERFORMANCE INCIDENT ALL CUSTOM"`
	Start      string `json:"start" validate:"required"`
	End        string `json:"end" validate:"required"`
}

func (h *Handlers) CreateReport(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostReportsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostReportsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	report, err := h.ReportService.CreateReport(
		ctx,
		req.TeamID,
		req.ReportType,
		entities.ReportData{},
	)
	if err != nil {
		c.Log.WithError(err).Error("failed to create report")

		return echo.ErrInternalServerError
	}

	err = h.EventService.Publish(events.ReportGenerateTask{
		ReportID:   report.ID,
		TeamID:     req.TeamID,
		ReportType: req.ReportType,
		Start:      req.Start,
		End:        req.End,
	})
	if err != nil {
		c.Log.WithError(err).Error("failed to publish report generation task")
		
		// Optional: mark report as failed since it couldn't be enqueued
		report.Status = entities.ReportStatusFailed
		_ = h.ReportService.Update(ctx, report)

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusCreated)
}
