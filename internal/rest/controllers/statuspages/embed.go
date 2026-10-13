package statuspages

import (
	"fmt"
	"html"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/statuspage"
)

var badgeStyles = map[statuspage.OverallStatus]struct{ label, color string }{
	statuspage.OverallStatusOperational: {"operational", "#10b981"},
	statuspage.OverallStatusOutage:      {"outage", "#f43f5e"},
	statuspage.OverallStatusMaintenance: {"maintenance", "#3b82f6"},
}

func renderBadge(status statuspage.OverallStatus) string {
	style, ok := badgeStyles[status]
	if !ok {
		style = struct{ label, color string }{"unknown", "#9ca3af"}
	}

	const left = "status"
	leftW := 7*len(left) + 12
	rightW := 7*len(style.label) + 12
	total := leftW + rightW

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%[1]d" height="20" role="img" aria-label="%[2]s: %[3]s">`+
		`<title>%[2]s: %[3]s</title>`+
		`<clipPath id="r"><rect width="%[1]d" height="20" rx="3" fill="#fff"/></clipPath>`+
		`<g clip-path="url(#r)"><rect width="%[4]d" height="20" fill="#555"/><rect x="%[4]d" width="%[5]d" height="20" fill="%[6]s"/></g>`+
		`<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">`+
		`<text x="%[7]d" y="14">%[2]s</text><text x="%[8]d" y="14">%[3]s</text></g></svg>`,
		total, left, html.EscapeString(style.label), leftW, rightW, style.color, leftW/2, leftW+rightW/2,
	)
}

func (h *Handlers) GetBadge(c echo.Context) error {
	page, err := h.loadPage(c, true)
	if err != nil {
		return err
	}

	status, err := h.StatusPageService.GetOverallStatus(c.Request().Context(), page)
	if err != nil {
		h.Logger.WithError(err).Error("failed to get status page status")

		return echo.ErrInternalServerError
	}

	c.Response().Header().Set("Cache-Control", "public, max-age=60")

	return c.Blob(http.StatusOK, "image/svg+xml; charset=utf-8", []byte(renderBadge(status)))
}

// widgetScript renders a small floating status indicator linking to the status page.
// %[1]q is the JSON endpoint, %[2]q the status page URL.
const widgetScript = `(function () {
  var api = %[1]q, link = %[2]q;
  var colors = { OPERATIONAL: "#10b981", OUTAGE: "#f43f5e", MAINTENANCE: "#3b82f6" };
  var texts = { OPERATIONAL: "All systems operational", OUTAGE: "Some systems are experiencing issues", MAINTENANCE: "Maintenance in progress" };
  fetch(api).then(function (r) { if (!r.ok) throw new Error(r.status); return r.json(); }).then(function (d) {
    var status = "OPERATIONAL";
    if ((d.activeIncidents || []).length || (d.monitors || []).some(function (m) { return m.status === "OUTAGE"; })) status = "OUTAGE";
    else if ((d.activeMaintenance || []).length) status = "MAINTENANCE";
    var a = document.createElement("a");
    a.href = link; a.target = "_blank"; a.rel = "noopener";
    a.style.cssText = "position:fixed;bottom:16px;right:16px;z-index:2147483647;display:flex;align-items:center;gap:8px;padding:8px 12px;border-radius:999px;background:#1e293b;color:#fff;font:13px/1.2 system-ui,sans-serif;text-decoration:none;box-shadow:0 4px 16px rgba(0,0,0,.25)";
    var dot = document.createElement("span");
    dot.style.cssText = "width:10px;height:10px;border-radius:50%%;background:" + colors[status];
    a.appendChild(dot);
    a.appendChild(document.createTextNode(texts[status]));
    (document.body || document.documentElement).appendChild(a);
  }).catch(function () {});
})();
`

func (h *Handlers) GetWidget(c echo.Context) error {
	page, err := h.loadPage(c, false)
	if err != nil {
		return err
	}

	scheme := "https"
	if c.Request().TLS == nil && c.Request().Header.Get("X-Forwarded-Proto") == "" {
		scheme = c.Scheme()
	}

	api := fmt.Sprintf("%s://%s/v1/public/status-pages/%s", scheme, c.Request().Host, page.Domain)
	js := fmt.Sprintf(widgetScript, api, "https://"+page.Domain)

	c.Response().Header().Set("Cache-Control", "public, max-age=300")

	return c.Blob(http.StatusOK, "application/javascript; charset=utf-8", []byte(js))
}
