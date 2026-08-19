package templates

import (
	_ "embed"
	"fmt"
)

//go:embed maintenance_alert.hbs
var MaintenanceAlertTemplateSource string

type MaintenanceAlertTemplate struct {
	BaseTemplate

	StatusPageName   string
	MaintenanceTitle string
	StatusPageURL    string
}

func (t *MaintenanceAlertTemplate) Subject() string {
	return fmt.Sprintf("Scheduled Maintenance: %s", t.StatusPageName)
}

func (t *MaintenanceAlertTemplate) HTML() string {
	return t.Render(MaintenanceAlertTemplateSource, map[string]any{
		"title":              "Scheduled Maintenance",
		"status_page_name":   t.StatusPageName,
		"maintenance_title":  t.MaintenanceTitle,
		"status_page_url":    t.StatusPageURL,
	})
}

func (t *MaintenanceAlertTemplate) PlainText() string {
	return fmt.Sprintf(`
Hi there,

This is a notification regarding scheduled maintenance for "%s":
%s

Please visit our status page for more details: %s
	`,
		t.StatusPageName,
		t.MaintenanceTitle,
		t.StatusPageURL,
	)
}
