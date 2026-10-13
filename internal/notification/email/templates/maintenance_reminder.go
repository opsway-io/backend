package templates

import (
	_ "embed"
	"fmt"
)

//go:embed maintenance_reminder.hbs
var MaintenanceReminderTemplateSource string

type MaintenanceReminderTemplate struct {
	BaseTemplate

	StatusPageName   string
	MaintenanceTitle string
	StatusPageURL    string
	UnsubscribeURL   string
}

func (t *MaintenanceReminderTemplate) Subject() string {
	return fmt.Sprintf("Reminder: Upcoming Maintenance for %s", t.StatusPageName)
}

func (t *MaintenanceReminderTemplate) HTML() string {
	return t.Render(MaintenanceReminderTemplateSource, map[string]any{
		"title":             "Upcoming Maintenance Reminder",
		"status_page_name":  t.StatusPageName,
		"maintenance_title": t.MaintenanceTitle,
		"status_page_url":   t.StatusPageURL,
		"unsubscribe_url":   t.UnsubscribeURL,
	})
}

func (t *MaintenanceReminderTemplate) PlainText() string {
	return fmt.Sprintf(`
Hi there,

This is a reminder that scheduled maintenance for "%s" will be starting soon.

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
