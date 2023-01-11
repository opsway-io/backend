package email

import (
	"context"
	"fmt"

	"github.com/opsway-io/backend/internal/notification/email/templates"
)

type ConsoleSender struct{}

// Prints email to console. Useful for development.
func NewConsoleSender() Sender {
	return &ConsoleSender{}
}

//nolint:forbidigo
func (s *ConsoleSender) Send(ctx context.Context, name string, to string, template templates.Template) error {
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("To: %s <%s>\n", name, to)
	fmt.Printf("Subject: %s\n", template.Subject())
	fmt.Println(template.PlainText())
	fmt.Println("------------------------------------------------------------")

	return nil
}
