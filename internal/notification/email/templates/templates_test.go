package templates

import (
	"strings"
	"testing"
)

func TestAllTemplatesRender(t *testing.T) {
	testCases := []struct {
		name     string
		tpl      Template
		expected string
	}{
		{
			name: "BillingAlertTemplate",
			tpl: &BillingAlertTemplate{
				TeamName:   "Opsway",
				Message:    "Card declined",
				BillingURL: "https://app.opsway.eu/billing",
			},
			expected: "Card declined",
		},
		{
			name: "DomainExpiryTemplate",
			tpl: &DomainExpiryTemplate{
				MonitorName:    "API",
				MonitorURL:     "https://api.opsway.eu",
				DashboardURL:   "https://app.opsway.eu",
				DaysRemaining:  7,
				ExpirationDate: "2026-11-01",
			},
			expected: "api.opsway.eu",
		},
		{
			name: "IncidentAlertTemplate",
			tpl: &IncidentAlertTemplate{
				Name:          "Alice",
				MonitorName:   "API",
				IncidentTitle: "Target Down",
				DashboardURL:  "https://app.opsway.eu",
			},
			expected: "Target Down",
		},
		{
			name: "MaintenanceAlertTemplate",
			tpl: &MaintenanceAlertTemplate{
				StatusPageName:   "Status",
				MaintenanceTitle: "DB Upgrade",
				StatusPageURL:    "https://status.opsway.eu",
				UnsubscribeURL:   "https://status.opsway.eu/unsub",
			},
			expected: "DB Upgrade",
		},
		{
			name: "MaintenanceConcludedTemplate",
			tpl: &MaintenanceConcludedTemplate{
				StatusPageName:   "Status",
				MaintenanceTitle: "DB Upgrade",
				StatusPageURL:    "https://status.opsway.eu",
				UnsubscribeURL:   "https://status.opsway.eu/unsub",
			},
			expected: "DB Upgrade",
		},
		{
			name: "MaintenanceReminderTemplate",
			tpl: &MaintenanceReminderTemplate{
				StatusPageName:   "Status",
				MaintenanceTitle: "DB Upgrade",
				StatusPageURL:    "https://status.opsway.eu",
				UnsubscribeURL:   "https://status.opsway.eu/unsub",
			},
			expected: "DB Upgrade",
		},
		{
			name: "NewUserWelcomeTemplate",
			tpl: &NewUserWelcomeTemplate{
				Name: "Alice",
			},
			expected: "Alice",
		},
		{
			name: "PasswordResetTemplate",
			tpl: &PasswordResetTemplate{
				Name:              "Alice",
				PasswordResetLink: "https://app.opsway.eu/reset",
			},
			expected: "https://app.opsway.eu/reset",
		},
		{
			name: "PerformanceDegradationTemplate",
			tpl: &PerformanceDegradationTemplate{
				MonitorName:    "Status Page",
				CurrentLatency: "Unexpected Spike",
				Threshold:      "Normal Baseline",
				DashboardURL:   "https://app.opsway.eu/incidents/1",
			},
			expected: "Unexpected Spike",
		},
		{
			name: "PostMortemPublishedTemplate",
			tpl: &PostMortemPublishedTemplate{
				StatusPageName: "Status",
				IncidentTitle:  "Outage",
				StatusPageURL:  "https://status.opsway.eu",
				UnsubscribeURL: "https://status.opsway.eu/unsub",
			},
			expected: "Outage",
		},
		{
			name: "RoleChangedTemplate",
			tpl: &RoleChangedTemplate{
				UserName:     "Alice",
				NewRole:      "ADMIN",
				TeamName:     "Opsway",
				DashboardURL: "https://app.opsway.eu",
			},
			expected: "ADMIN",
		},
		{
			name: "SSLExpiryTemplate",
			tpl: &SSLExpiryTemplate{
				MonitorName:    "API",
				MonitorURL:     "https://api.opsway.eu",
				DashboardURL:   "https://app.opsway.eu",
				DaysRemaining:  14,
				ExpirationDate: "2026-11-05",
			},
			expected: "api.opsway.eu",
		},
		{
			name: "StatuspageVerifyTemplate",
			tpl: &StatuspageVerifyTemplate{
				StatusPageName:  "Status",
				VerificationURL: "https://status.opsway.eu/verify",
			},
			expected: "https://status.opsway.eu/verify",
		},
		{
			name: "SubscriberIncidentAlertTemplate",
			tpl: &SubscriberIncidentAlertTemplate{
				StatusPageName: "Status",
				IncidentTitle:  "API Down",
				StatusPageURL:  "https://status.opsway.eu",
				IsResolved:     false,
				UnsubscribeURL: "https://status.opsway.eu/unsub",
			},
			expected: "API Down",
		},
		{
			name: "SubscriptionConfirmedTemplate",
			tpl: &SubscriptionConfirmedTemplate{
				StatusPageName: "Status",
				StatusPageURL:  "https://status.opsway.eu",
				UnsubscribeURL: "https://status.opsway.eu/unsub",
			},
			expected: "Status",
		},
		{
			name: "TeamInvitationTemplate",
			tpl: &TeamInvitationTemplate{
				TeamName:       "Opsway",
				ActivationLink: "https://app.opsway.eu/invite",
			},
			expected: "https://app.opsway.eu/invite",
		},
		{
			name: "WeeklyReportTemplate",
			tpl: &WeeklyReportTemplate{
				TeamName:          "Opsway",
				StartDate:         "2026-10-15",
				EndDate:           "2026-10-22",
				DashboardURL:      "https://app.opsway.eu",
				TotalMonitors:     4,
				MonitorsUp:        4,
				MonitorsDown:      0,
				TotalIncidents:    2,
				ResolvedIncidents: 2,
				AverageUptime:     "99.9",
			},
			expected: "99.9%",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			html := tc.tpl.HTML()
			if !strings.Contains(html, tc.expected) {
				t.Fatalf("expected HTML to contain %q, got:\n%s", tc.expected, html)
			}
			if tc.tpl.Subject() == "" {
				t.Fatalf("expected non-empty Subject()")
			}
			if tc.tpl.PlainText() == "" {
				t.Fatalf("expected non-empty PlainText()")
			}
		})
	}
}
