package templates

import (
	_ "embed"
	"fmt"
)

//go:embed role_changed.hbs
var roleChangedTemplate string

type RoleChangedTemplate struct {
	BaseTemplate
	UserName     string
	NewRole      string
	TeamName     string
	DashboardURL string
}

func (t *RoleChangedTemplate) Subject() string {
	return fmt.Sprintf("Notice: Your role in %s has been changed", t.TeamName)
}

func (t *RoleChangedTemplate) HTML() string {
	return t.Render(roleChangedTemplate, map[string]any{
		"UserName":     t.UserName,
		"NewRole":      t.NewRole,
		"TeamName":     t.TeamName,
		"DashboardURL": t.DashboardURL,
	})
}

func (t *RoleChangedTemplate) PlainText() string {
	return "Role Changed Alert"
}
