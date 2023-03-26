package templates

import (
	_ "embed"
	"fmt"
)

//go:embed password_reset.hbs
var PasswordResetTemplateSource string

type PasswordResetTemplate struct {
	BaseTemplate

	Name              string
	PasswordResetLink string
}

func (t *PasswordResetTemplate) Subject() string {
	return fmt.Sprintf("Password reset")
}

func (t *PasswordResetTemplate) HTML() string {
	return t.Render(PasswordResetTemplateSource, map[string]any{
		"title":               "Password reset",
		"name":                t.Name,
		"password_reset_link": t.PasswordResetLink,
	})
}

func (t *PasswordResetTemplate) PlainText() string {
	return fmt.Sprintf(`
Hi {{name}},
Somebody (hopefully you) has requested a password reset on your opsway account.
To reset your password, visit this link: %s
	`,
		t.PasswordResetLink,
	)
}
