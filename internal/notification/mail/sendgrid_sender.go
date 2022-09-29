package mail

import (
	"fmt"
	"net/http"

	"github.com/opsway-io/backend/internal/notification/mail/templates"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type SendConfig struct {
	SendgridAPIKey string
	SenderName     string
	SenderEmail    string
}

type SendgridSender struct {
	config SendConfig
	client *sendgrid.Client
}

func NewSendgridSender(config SendConfig) Sender {
	return &SendgridSender{
		config: config,
		client: sendgrid.NewSendClient(config.SendgridAPIKey),
	}
}

func (s *SendgridSender) Send(name string, to string, template templates.Template) error {
	sender := mail.NewEmail(s.config.SenderName, s.config.SenderEmail)
	receiver := mail.NewEmail(name, to)

	message := mail.NewSingleEmail(sender, template.Subject(), receiver, template.PlainText(), template.HTML())

	response, err := s.client.Send(message)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("sendgrid returned non-200 status code %d", response.StatusCode)
	}

	return nil
}
