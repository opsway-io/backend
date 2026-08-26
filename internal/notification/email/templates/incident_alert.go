package templates

import (
	_ "embed"
	"fmt"
)

//go:embed incident_alert.hbs
var IncidentAlertTemplateSource string

type IncidentAlertTemplate struct {
	BaseTemplate

	Name          string
	MonitorName   string
	IncidentTitle string
	DashboardURL  string
}

func (t *IncidentAlertTemplate) Subject() string {
	return fmt.Sprintf("Alert: Incident for %s", t.MonitorName)
}

func (t *IncidentAlertTemplate) HTML() string {
	return t.Render(IncidentAlertTemplateSource, map[string]any{
		"title":          "Monitor Alert",
		"name":           t.Name,
		"monitor_name":   t.MonitorName,
		"incident_title": t.IncidentTitle,
		"dashboard_url":  t.DashboardURL,
	})
}

func (t *IncidentAlertTemplate) PlainText() string {
	return fmt.Sprintf(`
Hi %s,

An incident has been triggered for monitor "%s":
Issue: %s

View the incident dashboard here: %s
	`,
		t.Name,
		t.MonitorName,
		t.IncidentTitle,
		t.DashboardURL,
	)
}
