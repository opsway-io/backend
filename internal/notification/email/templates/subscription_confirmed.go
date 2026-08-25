package templates

import (
	_ "embed"
	"fmt"
)

//go:embed subscription_confirmed.hbs
var SubscriptionConfirmedTemplateSource string

type SubscriptionConfirmedTemplate struct {
	BaseTemplate

	StatusPageName string
	StatusPageURL  string
	UnsubscribeURL string
}

func (t *SubscriptionConfirmedTemplate) Subject() string {
	return fmt.Sprintf("Subscription Confirmed: %s", t.StatusPageName)
}

func (t *SubscriptionConfirmedTemplate) HTML() string {
	return t.Render(SubscriptionConfirmedTemplateSource, map[string]any{
		"title":            "Subscription Confirmed",
		"status_page_name": t.StatusPageName,
		"status_page_url":  t.StatusPageURL,
		"unsubscribe_url":  t.UnsubscribeURL,
	})
}

func (t *SubscriptionConfirmedTemplate) PlainText() string {
	return fmt.Sprintf(`
Hi there,

Your subscription to "%s" has been confirmed!
You will now receive email notifications when incidents are created, updated, or resolved, as well as when maintenance is scheduled.

View the status page: %s

If you wish to stop receiving these emails, you can unsubscribe at any time:
%s
	`,
		t.StatusPageName,
		t.StatusPageURL,
		t.UnsubscribeURL,
	)
}
