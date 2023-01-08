package templates

import (
	_ "embed"

	"github.com/aymerick/raymond"
)

//go:embed base.hbs
var baseTemplateSource string

type BaseTemplate struct {
	subject string
	body    string
}

func (t *BaseTemplate) Render(source string, ctx map[string]any) string {
	content := raymond.MustRender(source, ctx)

	title, ok := ctx["title"].(string)
	if !ok {
		title = ""
	}

	return raymond.MustRender(baseTemplateSource, map[string]string{
		"title":   title,
		"content": content,
	})
}
