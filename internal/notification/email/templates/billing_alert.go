package templates

import (
	_ "embed"
	"fmt"
)

//go:embed billing_alert.hbs
var billingAlertTemplate string

type BillingAlertTemplate struct {
	BaseTemplate
	TeamName     string
	Message      string
	BillingURL   string
}

func (t *BillingAlertTemplate) Subject() string {
	return fmt.Sprintf("Action Required: Billing Alert for %s", t.TeamName)
}

func (t *BillingAlertTemplate) HTML() string {
	return t.Render(billingAlertTemplate, map[string]any{
		"TeamName":   t.TeamName,
		"Message":    t.Message,
		"BillingURL": t.BillingURL,
	})
}

func (t *BillingAlertTemplate) PlainText() string {
	return "Billing Alert: " + t.Message
}
