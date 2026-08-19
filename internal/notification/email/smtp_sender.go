package email

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/opsway-io/backend/internal/notification/email/templates"
)

type SMTPSender struct {
	config Config
}

func NewSMTPSender(config Config) Sender {
	return &SMTPSender{
		config: config,
	}
}

func (s *SMTPSender) Send(ctx context.Context, name string, to string, template templates.Template) error {
	from := s.config.SenderEmail
	if s.config.SenderName != "" {
		from = fmt.Sprintf("%s <%s>", s.config.SenderName, s.config.SenderEmail)
	}

	address := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)
	
	subject := template.Subject()
	body := template.HTML()
	
	msg := []byte("To: " + to + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	var auth smtp.Auth
	if s.config.SMTPUsername != "" || s.config.SMTPPassword != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}

	err := smtp.SendMail(address, auth, s.config.SenderEmail, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send SMTP email: %w", err)
	}

	return nil
}
