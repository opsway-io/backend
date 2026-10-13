package templates

import _ "embed"

//go:embed ssl_expiry.hbs
var sslExpiryTemplate string

type SSLExpiryTemplate struct {
	BaseTemplate
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
	return t.Render(sslExpiryTemplate, map[string]any{
		"MonitorName":    t.MonitorName,
		"MonitorURL":     t.MonitorURL,
		"DashboardURL":   t.DashboardURL,
		"DaysRemaining":  t.DaysRemaining,
		"ExpirationDate": t.ExpirationDate,
	})
}

func (t *SSLExpiryTemplate) PlainText() string {
	return "SSL Expiry Alert"
}
