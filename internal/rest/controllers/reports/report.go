package reports

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/report"
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
	ID        uint       `json:"id"`
	TeamID    uint       `json:"teamId"`
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	Start     *time.Time `json:"start,omitempty"`
	End       *time.Time `json:"end,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (h *Handlers) GetReports(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetReportsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetReportsRequest")

		return echo.ErrBadRequest
	}

	reports, err := h.ReportService.GetResportsByTeam(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get reports")

		return echo.ErrInternalServerError
	}

	resp := GetReportsResponse{
		Reports: make([]GetReportsResponseReport, len(*reports)),
	}
	for i, r := range *reports {
		resp.Reports[i] = newReportResponse(&r)
	}

	return c.JSON(http.StatusOK, resp)
}

func newReportResponse(r *entities.Report) GetReportsResponseReport {
	return GetReportsResponseReport{
		ID:        r.ID,
		TeamID:    r.TeamID,
		Type:      string(r.Type),
		Status:    string(r.Status),
		Start:     r.Start,
		End:       r.End,
		CreatedAt: r.CreatedAt,
	}
}

type GetReportRequest struct {
	TeamID   uint `param:"teamId" validate:"required,numeric,gte=0"`
	ReportID uint `param:"reportId" validate:"required,numeric,gte=0"`
}

type GetReportResponse struct {
	GetReportsResponseReport
	Data entities.ReportData `json:"data"`
}

func (h *Handlers) GetReport(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetReportRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetReportRequest")

		return echo.ErrBadRequest
	}

	r, err := h.ReportService.GetByTeamIDAndID(c.Request().Context(), req.TeamID, req.ReportID)
	if err != nil {
		if errors.Is(err, report.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to get report")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, GetReportResponse{
		GetReportsResponseReport: newReportResponse(r),
		Data:                     r.Report.Data(),
	})
}

type PostReportsRequest struct {
	TeamID     uint      `param:"teamId" validate:"required,numeric,gte=0"`
	ReportType string    `json:"reportType" validate:"required,oneof=UPTIME PERFORMANCE INCIDENT ALL CUSTOM"`
	Start      time.Time `json:"start" validate:"required"`
	End        time.Time `json:"end" validate:"required"`
}

func (h *Handlers) CreateReport(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostReportsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostReportsRequest")

		return echo.ErrBadRequest
	}

	if !req.End.After(req.Start) {
		return echo.NewHTTPError(http.StatusBadRequest, "end must be after start")
	}

	reportType, err := entities.ReportFrom(req.ReportType)
	if err != nil {
		return echo.ErrBadRequest
	}

	r, err := h.ReportService.Generate(c.Request().Context(), req.TeamID, reportType, req.Start, req.End)
	if err != nil {
		// The report is still stored (status FAILED) when only data collection failed
		c.Log.WithError(err).Error("failed to generate report")

		if r == nil {
			return echo.ErrInternalServerError
		}
	}

	return c.JSON(http.StatusCreated, newReportResponse(r))
}

type DeleteReportRequest struct {
	TeamID   uint `param:"teamId" validate:"required,numeric,gte=0"`
	ReportID uint `param:"reportId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteReport(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteReportRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	if err := h.ReportService.Delete(c.Request().Context(), req.TeamID, req.ReportID); err != nil {
		if errors.Is(err, report.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to delete report")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
