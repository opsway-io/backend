package mail

import (
	"fmt"

	"github.com/opsway-io/backend/internal/notification/mail/templates"
)

type DummySender struct{}

func NewDummySender() Sender {
	return &DummySender{}
}

//nolint:forbidigo
func (s *DummySender) Send(name string, to string, template templates.Template) error {
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("To: %s <%s>\n", name, to)
	fmt.Printf("Subject: %s\n", template.Subject())
	fmt.Println(template.PlainText())
	fmt.Println("------------------------------------------------------------")

	return nil
}
