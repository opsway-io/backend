package templates

import (
	"fmt"
)

const incidentAlertTemplateSource = `<p>Hi {{name}},</p>
<p>The alert rule <strong>{{rule_name}}</strong> was triggered by a new incident:</p>
<p><strong>{{incident_title}}</strong></p>
<p>{{incident_description}}</p>
<p><a href="{{incident_link}}">View incident</a></p>`

type IncidentAlertTemplate struct {
	BaseTemplate

	Name                string
	RuleName            string
	IncidentTitle       string
	IncidentDescription string
	IncidentLink        string
}

func (t *IncidentAlertTemplate) Subject() string {
	return fmt.Sprintf("[opsway] Incident: %s", t.IncidentTitle)
}

func (t *IncidentAlertTemplate) HTML() string {
	return t.Render(incidentAlertTemplateSource, map[string]any{
		"title":                "New incident",
		"name":                 t.Name,
		"rule_name":            t.RuleName,
		"incident_title":       t.IncidentTitle,
		"incident_description": t.IncidentDescription,
		"incident_link":        t.IncidentLink,
	})
}

func (t *IncidentAlertTemplate) PlainText() string {
	return fmt.Sprintf(
		"Hi %s,\nThe alert rule %q was triggered by a new incident: %s\n%s\nView incident: %s\n",
		t.Name, t.RuleName, t.IncidentTitle, t.IncidentDescription, t.IncidentLink,
	)
}
