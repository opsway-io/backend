package user

import "time"

type Config struct {
	PasswordResetTokenTTL time.Duration `mapstructure:"password_reset_token_ttl" default:"24h"`
	PasswordResetURL      string        `mapstructure:"password_reset_url"`
}
