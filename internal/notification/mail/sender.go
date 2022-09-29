package mail

import (
	"github.com/opsway-io/backend/internal/notification/mail/templates"
)

type Sender interface {
	Send(name string, to string, template templates.Template) error
}
