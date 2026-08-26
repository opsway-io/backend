package templates

import (
	_ "embed"
	"fmt"
)

//go:embed post_mortem_published.hbs
var PostMortemPublishedTemplateSource string

type PostMortemPublishedTemplate struct {
	BaseTemplate

	StatusPageName string
	IncidentTitle  string
	StatusPageURL  string
	UnsubscribeURL string
}

func (t *PostMortemPublishedTemplate) Subject() string {
	return fmt.Sprintf("Post-Mortem Published: %s", t.StatusPageName)
}

func (t *PostMortemPublishedTemplate) HTML() string {
	return t.Render(PostMortemPublishedTemplateSource, map[string]any{
		"title":            "Post-Mortem Published",
		"status_page_name": t.StatusPageName,
		"incident_title":   t.IncidentTitle,
		"status_page_url":  t.StatusPageURL,
		"unsubscribe_url":  t.UnsubscribeURL,
	})
}

func (t *PostMortemPublishedTemplate) PlainText() string {
	return fmt.Sprintf(`
Hi there,

A post-mortem has been published for the incident "%s" on "%s".

Please visit our status page to read the full details: %s

If you wish to stop receiving these emails, you can unsubscribe here:
%s
	`,
		t.IncidentTitle,
		t.StatusPageName,
		t.StatusPageURL,
		t.UnsubscribeURL,
	)
}
