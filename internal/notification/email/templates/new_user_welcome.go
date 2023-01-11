package templates

import (
	_ "embed"
	"fmt"
)

//go:embed new_user_welcome.hbs
var NewUserWelcomeTemplateSource string

type NewUserWelcomeTemplate struct {
	BaseTemplate

	Name string
}

func (t *NewUserWelcomeTemplate) Subject() string {
	return fmt.Sprintf("Welcome to opsway!")
}

func (t *NewUserWelcomeTemplate) HTML() string {
	return t.Render(NewUserWelcomeTemplateSource, map[string]any{
		"title": "Welcome to opsway!",
		"name":  t.Name,
	})
}

func (t *NewUserWelcomeTemplate) PlainText() string {
	return fmt.Sprintf("Hi %s, We're thrilled to have you here!", t.Name)
}
