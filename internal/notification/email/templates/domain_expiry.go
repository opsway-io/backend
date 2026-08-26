package templates

import _ "embed"

//go:embed domain_expiry.hbs
var domainExpiryTemplate string

type DomainExpiryTemplate struct {
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
	return renderTemplate(domainExpiryTemplate, t)
}
