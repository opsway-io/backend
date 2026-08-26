package templates

import (
	_ "embed"
	"fmt"
)

//go:embed maintenance_concluded.hbs
var MaintenanceConcludedTemplateSource string

type MaintenanceConcludedTemplate struct {
	BaseTemplate

	StatusPageName   string
	MaintenanceTitle string
	StatusPageURL    string
	UnsubscribeURL   string
}

func (t *MaintenanceConcludedTemplate) Subject() string {
	return fmt.Sprintf("Maintenance Concluded: %s", t.StatusPageName)
}

func (t *MaintenanceConcludedTemplate) HTML() string {
	return t.Render(MaintenanceConcludedTemplateSource, map[string]any{
		"title":             "Maintenance Concluded",
		"status_page_name":  t.StatusPageName,
		"maintenance_title": t.MaintenanceTitle,
		"status_page_url":   t.StatusPageURL,
		"unsubscribe_url":   t.UnsubscribeURL,
	})
}

func (t *MaintenanceConcludedTemplate) PlainText() string {
	return fmt.Sprintf(`
Hi there,

The scheduled maintenance for "%s" has concluded.

Maintenance Details:
%s

Please visit our status page for more details: %s

If you wish to stop receiving these emails, you can unsubscribe here:
%s
	`,
		t.StatusPageName,
		t.MaintenanceTitle,
		t.StatusPageURL,
		t.UnsubscribeURL,
	)
}
