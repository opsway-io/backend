package templates

import _ "embed"

//go:embed ssl_expiry.hbs
var sslExpiryTemplate string

type SSLExpiryTemplate struct {
	MonitorName    string
	MonitorURL     string
	DashboardURL   string
	DaysRemaining  int
	ExpirationDate string
}

func (t *SSLExpiryTemplate) Subject() string {
	return "⚠️ Action Required: SSL Certificate Expiring Soon for " + t.MonitorName
}

func (t *SSLExpiryTemplate) HTML() string {
	return renderTemplate(sslExpiryTemplate, t)
}
