package reports

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
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
	CreatedAt string `json:"createdAt"`
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

func (h *Handlers) newGetReportResponse(reports *[]entities.Report) *GetReportsResponse {
	resp := &GetReportsResponse{
		Reports: make([]GetReportsResponseReport, len(*reports)),
	}

	for i, in := range *reports {
		resp.Reports[i] = GetReportsResponseReport{
			ID:        in.ID,
			TeamID:    in.TeamID,
			CreatedAt: in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return resp
}

type PostReportsRequest struct {
	TeamID     uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ReportType string `json:"reportType" validate:"required,oneof=incident monitor"`
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

	uptimeReport, err := h.CheckService.GetByTeamIDMonitorsUptime(ctx, req.TeamID, req.Start, req.End)
	if err != nil {
		c.Log.WithError(err).Error("failed to get uptime report")
		return echo.ErrInternalServerError
	}

	err = h.ReportService.CreateReport(
		ctx,
		req.TeamID,
		req.ReportType,
		entities.ReportData{
			Uptime: uptimeReport,
		},
	)
	if err != nil {
		c.Log.WithError(err).Error("failed to get Reports")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusCreated)
}
