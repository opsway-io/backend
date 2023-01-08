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
	return fmt.Sprintf("Welcome to team %s", t.TeamName)
}

func (t *TeamInvitationTemplate) HTML() string {
	return t.Render(teamInvitationTemplateSource, map[string]any{
		"title":           "Welcome on board!",
		"activation_link": t.ActivationLink,
		"team_name":       t.TeamName,
	})
}

func (t *TeamInvitationTemplate) PlainText() string {
	return fmt.Sprintf(`
Welcome to team %s!

Please follow this link to activate your account: %s

Best regards,
Opsway team
	`, t.TeamName, t.ActivationLink)
}
