package templates

import (
	_ "embed"
	"fmt"
)

//go:embed team_invitation.hbs
var teamInvitationTemplateSource string

type TeamInvitationTemplate struct {
	BaseTemplate

	TeamName       string
	ActivationLink string
}

func (t *TeamInvitationTemplate) Subject() string {
	return fmt.Sprintf("Invitation to team %s", t.TeamName)
}

func (t *TeamInvitationTemplate) HTML() string {
	return t.Render(teamInvitationTemplateSource, map[string]any{
		"title":           fmt.Sprintf("Invitation to team %s", t.TeamName),
		"activation_link": t.ActivationLink,
		"team_name":       t.TeamName,
	})
}

func (t *TeamInvitationTemplate) PlainText() string {
	return fmt.Sprintf(`
You have been invited to join team %s!

Visit the following link to accept the invite: %s
`, t.TeamName, t.ActivationLink)
}
