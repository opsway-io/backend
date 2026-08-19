package templates

import (
	_ "embed"
	"fmt"
)

//go:embed subscriber_incident_alert.hbs
var SubscriberIncidentAlertTemplateSource string

type SubscriberIncidentAlertTemplate struct {
	BaseTemplate

	StatusPageName string
	IncidentTitle  string
	StatusPageURL  string
	IsResolved     bool
}

func (t *SubscriberIncidentAlertTemplate) Subject() string {
	if t.IsResolved {
		return fmt.Sprintf("Resolved: Incident for %s", t.StatusPageName)
	}
	return fmt.Sprintf("Alert: Incident for %s", t.StatusPageName)
}

func (t *SubscriberIncidentAlertTemplate) HTML() string {
	titleText := "Service Alert"
	if t.IsResolved {
		titleText = "Service Restored"
	}

	return t.Render(SubscriberIncidentAlertTemplateSource, map[string]any{
		"title":            titleText,
		"status_page_name": t.StatusPageName,
		"incident_title":   t.IncidentTitle,
		"status_page_url":  t.StatusPageURL,
		"is_resolved":      t.IsResolved,
	})
}

func (t *SubscriberIncidentAlertTemplate) PlainText() string {
	status := "An incident has been reported"
	if t.IsResolved {
		status = "An incident has been resolved"
	}

	return fmt.Sprintf(`
Hi there,

%s for "%s":
%s

View the latest updates on our status page: %s
	`,
		status,
		t.StatusPageName,
		t.IncidentTitle,
		t.StatusPageURL,
	)
}
