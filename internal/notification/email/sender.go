package email

import (
	"context"

	"github.com/opsway-io/backend/internal/notification/email/templates"
)

type Config struct {
	Debug          bool
	SenderName     string `mapstructure:"sender_name"`
	SenderEmail    string `mapstructure:"sender_email"`
	SendgridAPIKey string `mapstructure:"sendgrid_api_key"`
}

type Sender interface {
	Send(ctx context.Context, name string, to string, template templates.Template) error
}
