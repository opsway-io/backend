package reports

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetReportsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
	Offset *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit  *int `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetReportsResponse struct {
	Reports []GetReportsResponseReport `json:"reports"`
}

type GetReportsResponseReport struct {
	ID          uint   `json:"id"`
	TeamID      uint   `json:"teamId"`
	MonitorID   uint   `json:"monitorId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
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

	// resp := h.newGetReportResponse(reports)

	return c.JSON(http.StatusOK, len(reports))
}

// func (h *Handlers) newGetReportResponse(reports *[]entities.Report) *GetReportsResponse {
// 	resp := &GetReportsResponse{
// 		Reports: make([]GetReportsResponseReport, len(*reports)),
// 	}

// 	for i, in := range *reports {
// 		resp.Reports[i] = GetReportsResponseReport{
// 			ID:          in.ID,
// 			TeamID:      in.TeamID,
// 			MonitorID:   in.MonitorID,
// 			Title:       in.Title,
// 			Description: *in.Description,
// 			CreatedAt:   in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
// 		}
// 	}

// 	return resp
// }
