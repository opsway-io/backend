package statuspages

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/sirupsen/logrus"
)

// ComputeOverallStatus returns ("OPERATIONAL"|"DEGRADED"|"OUTAGE"|"MAINTENANCE")
func (h *PublicHandlers) ComputeOverallStatus(c echo.Context, domain string) (string, error) {
	sp, err := h.StatusPageService.GetByDomain(c.Request().Context(), domain)
	if err != nil {
		return "", err
	}

	uniqueMonitors := make(map[uint]bool)
	var monitorIDs []uint
	for _, m := range sp.Monitors {
		if !uniqueMonitors[m.ID] {
			uniqueMonitors[m.ID] = true
			monitorIDs = append(monitorIDs, m.ID)
		}
	}
	for _, g := range sp.Groups {
		for _, m := range g.Monitors {
			if !uniqueMonitors[m.ID] {
				uniqueMonitors[m.ID] = true
				monitorIDs = append(monitorIDs, m.ID)
			}
		}
	}

	if len(monitorIDs) == 0 {
		return "OPERATIONAL", nil
	}

	// Check active maintenance first
	if h.MaintenanceService != nil {
		maintenances, err := h.MaintenanceService.GetActiveByMonitorIDs(c.Request().Context(), time.Now(), monitorIDs)
		if err == nil && len(maintenances) > 0 {
			return "MAINTENANCE", nil
		}
	}

	// Check active incidents
	if h.IncidentService != nil {
		incidents, err := h.IncidentService.GetActiveByMonitorIDs(c.Request().Context(), monitorIDs)
		if err == nil && len(incidents) > 0 {
			if len(incidents) >= len(monitorIDs) {
				return "OUTAGE", nil
			}
			return "DEGRADED", nil
		}
	}

	return "OPERATIONAL", nil
}

func (h *PublicHandlers) GetBadgeSVG(c echo.Context) error {
	domain := c.Param("domain")
	if domain == "" {
		return echo.ErrBadRequest
	}

	status, err := h.ComputeOverallStatus(c, domain)
	if err != nil {
		if errors.Is(err, statuspage.ErrNotFound) {
			return echo.ErrNotFound
		}
		logrus.WithError(err).Error("failed to compute overall status for badge")
		return echo.ErrInternalServerError
	}

	label := c.QueryParam("label")
	if label == "" {
		label = "status"
	}

	style := c.QueryParam("style")

	statusText := "operational"
	statusColor := "#4caf50" // green

	switch status {
	case "MAINTENANCE":
		statusText = "maintenance"
		statusColor = "#2196f3" // blue
	case "DEGRADED":
		statusText = "degraded"
		statusColor = "#ff9800" // orange
	case "OUTAGE":
		statusText = "outage"
		statusColor = "#f44336" // red
	}

	radius := "3"
	if style == "flat-square" {
		radius = "0"
	}

	labelWidth := len(label)*7 + 10
	statusWidth := len(statusText)*7 + 10
	totalWidth := labelWidth + statusWidth
	labelMid := labelWidth / 2
	statusMid := labelWidth + (statusWidth / 2)

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">
  <linearGradient id="b" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="a">
    <rect width="%d" height="20" rx="%s" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#a)">
    <rect width="%d" height="20" fill="#555"/>
    <rect x="%d" width="%d" height="20" fill="%s"/>
    <rect width="%d" height="20" fill="url(#b)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">
    <text aria-hidden="true" x="%d0" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d0">%s</text>
    <text x="%d0" y="140" transform="scale(.1)" fill="#fff" textLength="%d0">%s</text>
    <text aria-hidden="true" x="%d0" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d0">%s</text>
    <text x="%d0" y="140" transform="scale(.1)" fill="#fff" textLength="%d0">%s</text>
  </g>
</svg>`,
		totalWidth, label, statusText,
		totalWidth, radius,
		labelWidth, labelWidth, statusWidth, statusColor, totalWidth,
		labelMid, (labelWidth - 10), label,
		labelMid, (labelWidth - 10), label,
		statusMid, (statusWidth - 10), statusText,
		statusMid, (statusWidth - 10), statusText,
	)

	c.Response().Header().Set("Content-Type", "image/svg+xml")
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	return c.String(http.StatusOK, svg)
}

type WidgetSummaryResponse struct {
	Domain            string `json:"domain"`
	Status            string `json:"status"` // OPERATIONAL, DEGRADED, OUTAGE, MAINTENANCE
	StatusColor       string `json:"statusColor"`
	ActiveIncidents   int    `json:"activeIncidents"`
	ActiveMaintenance int    `json:"activeMaintenance"`
	StatusPageURL     string `json:"statusPageURL"`
}

func (h *PublicHandlers) GetWidgetSummary(c echo.Context) error {
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

	status, err := h.ComputeOverallStatus(c, domain)
	if err != nil {
		return echo.ErrInternalServerError
	}

	statusColor := "#4caf50"
	switch status {
	case "MAINTENANCE":
		statusColor = "#2196f3"
	case "DEGRADED":
		statusColor = "#ff9800"
	case "OUTAGE":
		statusColor = "#f44336"
	}

	statusPageURL := fmt.Sprintf("%s/%s", h.StatusPageBaseURL, sp.Domain)
	if h.StatusPageBaseURL == "" {
		statusPageURL = fmt.Sprintf("/status-pages/%s", sp.Domain)
	}

	return c.JSON(http.StatusOK, WidgetSummaryResponse{
		Domain:        domain,
		Status:        status,
		StatusColor:   statusColor,
		StatusPageURL: statusPageURL,
	})
}

func (h *PublicHandlers) GetWidgetScript(c echo.Context) error {
	domain := c.Param("domain")
	if domain == "" {
		return echo.ErrBadRequest
	}

	escapedDomain := strings.ReplaceAll(domain, `"`, "")

	script := fmt.Sprintf(`(function() {
  var domain = "%s";
  var apiBase = window.OPSWAY_API_URL || "";
  var summaryUrl = (apiBase ? apiBase : "") + "/v1/public/status-pages/" + encodeURIComponent(domain) + "/widget.json";

  function renderWidget(data) {
    if (!data || data.status === "OPERATIONAL") return;

    var bar = document.createElement("div");
    bar.id = "opsway-status-banner";
    bar.style.position = "fixed";
    bar.style.top = "0";
    bar.style.left = "0";
    bar.style.width = "100%%";
    bar.style.zIndex = "999999";
    bar.style.backgroundColor = data.statusColor || "#ff9800";
    bar.style.color = "#ffffff";
    bar.style.fontFamily = "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif";
    bar.style.fontSize = "14px";
    bar.style.fontWeight = "500";
    bar.style.padding = "10px 16px";
    bar.style.boxShadow = "0 2px 4px rgba(0,0,0,0.2)";
    bar.style.display = "flex";
    bar.style.alignItems = "center";
    bar.style.justifyContent = "space-between";

    var textSpan = document.createElement("span");
    textSpan.innerHTML = "<strong>Notice:</strong> We are currently experiencing service issues (" + data.status + ").";

    var link = document.createElement("a");
    link.href = data.statusPageURL;
    link.target = "_blank";
    link.rel = "noopener noreferrer";
    link.innerText = "View Status Page →";
    link.style.color = "#ffffff";
    link.style.textDecoration = "underline";
    link.style.marginLeft = "12px";

    var rightContainer = document.createElement("div");
    rightContainer.style.display = "flex";
    rightContainer.style.alignItems = "center";

    var closeBtn = document.createElement("button");
    closeBtn.innerHTML = "&times;";
    closeBtn.style.background = "none";
    closeBtn.style.border = "none";
    closeBtn.style.color = "#ffffff";
    closeBtn.style.fontSize = "18px";
    closeBtn.style.cursor = "pointer";
    closeBtn.style.marginLeft = "16px";
    closeBtn.onclick = function() { bar.remove(); };

    rightContainer.appendChild(link);
    rightContainer.appendChild(closeBtn);
    bar.appendChild(textSpan);
    bar.appendChild(rightContainer);

    document.body.prepend(bar);
  }

  fetch(summaryUrl)
    .then(function(res) { return res.json(); })
    .then(function(data) { renderWidget(data); })
    .catch(function(err) { console.error("Opsway widget error:", err); });
})();`, escapedDomain)

	c.Response().Header().Set("Content-Type", "application/javascript")
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	return c.String(http.StatusOK, script)
}
