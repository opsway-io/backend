package templates

import _ "embed"

//go:embed domain_expiry.hbs
var domainExpiryTemplate string

type DomainExpiryTemplate struct {
	BaseTemplate
	MonitorName    string
	MonitorURL     string
	DashboardURL   string
	DaysRemaining  int
	ExpirationDate string
}

func (t *DomainExpiryTemplate) Subject() string {
	return "⚠️ Action Required: Domain Expiring Soon for " + t.MonitorName
}

func (t *DomainExpiryTemplate) HTML() string {
	return t.Render(domainExpiryTemplate, map[string]any{
		"MonitorName":    t.MonitorName,
		"MonitorURL":     t.MonitorURL,
		"DashboardURL":   t.DashboardURL,
		"DaysRemaining":  t.DaysRemaining,
		"ExpirationDate": t.ExpirationDate,
	})
}

func (t *DomainExpiryTemplate) PlainText() string {
	return "Domain Expiry Alert"
}
