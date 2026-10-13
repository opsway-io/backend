package statuspages

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/statuspage"
)

func (h *PublicHandlers) GetMaintenanceCalendar(c echo.Context) error {
	domain := c.Param("domain")
	if domain == "" {
		return echo.ErrBadRequest
	}

	sp, err := h.StatusPageService.GetByDomain(c.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, statuspage.ErrNotFound) {
			return echo.ErrNotFound
		}
		return echo.ErrInternalServerError
	}

	var monitorIDs []uint
	unique := make(map[uint]bool)
	for _, m := range sp.Monitors {
		if !unique[m.ID] {
			unique[m.ID] = true
			monitorIDs = append(monitorIDs, m.ID)
		}
	}
	for _, g := range sp.Groups {
		for _, m := range g.Monitors {
			if !unique[m.ID] {
				unique[m.ID] = true
				monitorIDs = append(monitorIDs, m.ID)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//Opsway//Maintenance Calendar//EN\r\n")
	sb.WriteString(fmt.Sprintf("X-WR-CALNAME:%s - Scheduled Maintenance\r\n", sp.Name))
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")

	if h.MaintenanceService != nil && len(monitorIDs) > 0 {
		maintenances, err := h.MaintenanceService.GetAllByMonitorIDs(c.Request().Context(), monitorIDs)
		if err == nil {
			nowStr := time.Now().UTC().Format("20060102T150405Z")
			for _, mnt := range maintenances {
				start := mnt.Settings.StartAt.UTC().Format("20060102T150405Z")
				end := mnt.Settings.EndAt.UTC().Format("20060102T150405Z")
				desc := ""
				if mnt.Description != nil {
					desc = strings.ReplaceAll(*mnt.Description, "\n", "\\n")
				}

				sb.WriteString("BEGIN:VEVENT\r\n")
				sb.WriteString(fmt.Sprintf("UID:mnt-%d@%s\r\n", mnt.ID, sp.Domain))
				sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", nowStr))
				sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", start))
				sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", end))
				sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", mnt.Title))
				if desc != "" {
					sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", desc))
				}
				sb.WriteString("STATUS:CONFIRMED\r\n")
				sb.WriteString("END:VEVENT\r\n")
			}
		}
	}

	sb.WriteString("END:VCALENDAR\r\n")

	c.Response().Header().Set("Content-Type", "text/calendar; charset=utf-8")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-maintenance.ics\"", sp.Domain))
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	return c.String(http.StatusOK, sb.String())
}

type PublicMonitorMetrics struct {
	MonitorID        uint      `json:"monitorId"`
	Name             string    `json:"name"`
	UptimePercentage float32   `json:"uptimePercentage"`
	DailyUptimes     []float32 `json:"dailyUptimes"`
}

type PublicStatusMetricsResponse struct {
	Domain   string                 `json:"domain"`
	Monitors []PublicMonitorMetrics `json:"monitors"`
}

func (h *PublicHandlers) GetPublicMetrics(c echo.Context) error {
	domain := c.Param("domain")
	if domain == "" {
		return echo.ErrBadRequest
	}

	sp, err := h.StatusPageService.GetByDomain(c.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, statuspage.ErrNotFound) {
			return echo.ErrNotFound
		}
		return echo.ErrInternalServerError
	}

	uniqueMonitors := make(map[uint]string)
	for _, m := range sp.Monitors {
		uniqueMonitors[m.ID] = m.Name
	}
	for _, g := range sp.Groups {
		for _, m := range g.Monitors {
			uniqueMonitors[m.ID] = m.Name
		}
	}

	var monitorIDs []uint
	for id := range uniqueMonitors {
		monitorIDs = append(monitorIDs, id)
	}

	var resMonitors []PublicMonitorMetrics
	if h.CheckService != nil && len(monitorIDs) > 0 {
		uptimes, _ := h.CheckService.GetMonitorUptimesByMonitorIDs(c.Request().Context(), monitorIDs)
		for _, id := range monitorIDs {
			var uptimePct float32 = 100.0
			var daily []float32
			if u, ok := uptimes[id]; ok {
				uptimePct = u.UptimePercentage
				daily = u.DailyUptimes
			}
			resMonitors = append(resMonitors, PublicMonitorMetrics{
				MonitorID:        id,
				Name:             uniqueMonitors[id],
				UptimePercentage: uptimePct,
				DailyUptimes:     daily,
			})
		}
	}
	if resMonitors == nil {
		resMonitors = []PublicMonitorMetrics{}
	}

	return c.JSON(http.StatusOK, PublicStatusMetricsResponse{
		Domain:   domain,
		Monitors: resMonitors,
	})
}
