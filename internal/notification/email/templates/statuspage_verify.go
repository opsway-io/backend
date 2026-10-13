package templates

import (
	_ "embed"
	"fmt"
)

//go:embed statuspage_verify.hbs
var StatuspageVerifyTemplateSource string

type StatuspageVerifyTemplate struct {
	BaseTemplate

	StatusPageName  string
	VerificationURL string
}

func (t *StatuspageVerifyTemplate) Subject() string {
	return fmt.Sprintf("Confirm your subscription to %s updates", t.StatusPageName)
}

func (t *StatuspageVerifyTemplate) HTML() string {
	return t.Render(StatuspageVerifyTemplateSource, map[string]any{
		"title":            "Confirm Subscription",
		"status_page_name": t.StatusPageName,
		"verification_url": t.VerificationURL,
	})
}

func (t *StatuspageVerifyTemplate) PlainText() string {
	return fmt.Sprintf(`
Hi there,

Please confirm your subscription to receive status updates for "%s" by clicking the link below:

%s
	`,
		t.StatusPageName,
		t.VerificationURL,
	)
}
