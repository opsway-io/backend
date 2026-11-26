package billing

import (
	"net/url"
	"strings"

	"github.com/opsway-io/backend/internal/entities"
)

type CatalogPlanLimits struct {
	TeamMembers         int `json:"teamMembers"`
	Monitors            int `json:"monitors"`
	StatusPages         int `json:"statusPages"`
	LogSearchGB         int `json:"logSearchGb"`
	ManagedAgentRuns    int `json:"managedAgentRuns"`
	PantomathRepos      int `json:"pantomathRepos"`
}

type CatalogSuitePlan struct {
	ID          entities.EntitlementTier `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Price       int64                    `json:"price"`
	Currency    string                   `json:"currency"`
	Interval    string                   `json:"interval"`
	LookupKey   string                   `json:"lookupKey"`
	Highlighted bool                     `json:"highlighted,omitempty"`
	Features    []string                 `json:"features"`
	Limits      CatalogPlanLimits        `json:"limits"`
}

type CatalogAddonTier struct {
	Tier         entities.EntitlementTier `json:"tier"`
	Name         string                   `json:"name"`
	Price        int64                    `json:"price"`
	Currency     string                   `json:"currency"`
	LookupKey    string                   `json:"lookupKey"`
	MonthlyQuota int                      `json:"monthlyQuota"`
	QuotaUnit    string                   `json:"quotaUnit"`
}

type CatalogProductAddon struct {
	Product     entities.ProductSlug `json:"product"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Subdomain   string               `json:"subdomain"`
	Tiers       []CatalogAddonTier   `json:"tiers"`
}

type BillingCatalog struct {
	Currency string                `json:"currency"`
	Plans    []CatalogSuitePlan    `json:"plans"`
	Addons   []CatalogProductAddon `json:"addons"`
}

// DefaultBillingCatalog returns the single source of truth for Opsway's
// EUR suite plans and modular product add-ons.
func DefaultBillingCatalog() BillingCatalog {
	return BillingCatalog{
		Currency: "eur",
		Plans: []CatalogSuitePlan{
			{
				ID:          entities.EntitlementTierFree,
				Name:        "Free",
				Description: "For small teams & standalone product trials",
				Price:       0,
				Currency:    "eur",
				Interval:    "month",
				LookupKey:   "free",
				Features: []string{
					"Up to 3 team members",
					"5 monitors & 1 status page",
					"1 GB / month Log Search",
					"3 Managed AI investigations / mo",
					"1 Agent Context repository",
					"Unified MCP Gateway & BYOA",
				},
				Limits: CatalogPlanLimits{
					TeamMembers:      entities.DefaultTeamMemberLimit(entities.EntitlementTierFree),
					Monitors:         entities.DefaultMonthlyQuota(entities.ProductMonitoring, entities.EntitlementTierFree),
					StatusPages:      entities.DefaultStatusPageLimit(entities.EntitlementTierFree),
					LogSearchGB:      entities.DefaultMonthlyQuota(entities.ProductLogSearch, entities.EntitlementTierFree),
					ManagedAgentRuns: entities.DefaultMonthlyQuota(entities.ProductManagedAgent, entities.EntitlementTierFree),
					PantomathRepos:   entities.DefaultMonthlyQuota(entities.ProductPantomath, entities.EntitlementTierFree),
				},
			},
			{
				ID:          entities.EntitlementTierPro,
				Name:        "Pro",
				Description: "For growing engineering teams",
				Price:       20,
				Currency:    "eur",
				Interval:    "month",
				LookupKey:   "pro",
				Highlighted: true,
				Features: []string{
					"Up to 5 team members",
					"50 monitors & 5 status pages",
					"50 GB / month Log Search",
					"30 Managed AI investigations / mo",
					"5 Agent Context repositories",
					"Pay-as-you-go overage protection & 1-click remediation",
				},
				Limits: CatalogPlanLimits{
					TeamMembers:      entities.DefaultTeamMemberLimit(entities.EntitlementTierPro),
					Monitors:         entities.DefaultMonthlyQuota(entities.ProductMonitoring, entities.EntitlementTierPro),
					StatusPages:      entities.DefaultStatusPageLimit(entities.EntitlementTierPro),
					LogSearchGB:      entities.DefaultMonthlyQuota(entities.ProductLogSearch, entities.EntitlementTierPro),
					ManagedAgentRuns: entities.DefaultMonthlyQuota(entities.ProductManagedAgent, entities.EntitlementTierPro),
					PantomathRepos:   entities.DefaultMonthlyQuota(entities.ProductPantomath, entities.EntitlementTierPro),
				},
			},
			{
				ID:          entities.EntitlementTierEnterprise,
				Name:        "Enterprise",
				Description: "For organizations running full-suite operations",
				Price:       100,
				Currency:    "eur",
				Interval:    "month",
				LookupKey:   "enterprise",
				Features: []string{
					"Unlimited team members",
					"Unlimited monitors & status pages",
					"Unlimited Log Search ingestion",
					"500 Managed AI investigations / mo",
					"Unlimited Agent Context repositories",
					"Autonomous auto-remediation & priority 24/7 support",
				},
				Limits: CatalogPlanLimits{
					TeamMembers:      entities.DefaultTeamMemberLimit(entities.EntitlementTierEnterprise),
					Monitors:         entities.DefaultMonthlyQuota(entities.ProductMonitoring, entities.EntitlementTierEnterprise),
					StatusPages:      entities.DefaultStatusPageLimit(entities.EntitlementTierEnterprise),
					LogSearchGB:      entities.DefaultMonthlyQuota(entities.ProductLogSearch, entities.EntitlementTierEnterprise),
					ManagedAgentRuns: entities.DefaultMonthlyQuota(entities.ProductManagedAgent, entities.EntitlementTierEnterprise),
					PantomathRepos:   entities.DefaultMonthlyQuota(entities.ProductPantomath, entities.EntitlementTierEnterprise),
				},
			},
		},
		Addons: []CatalogProductAddon{
			{
				Product:     entities.ProductMonitoring,
				Name:        "Opsway Monitoring",
				Description: "Multi-location probes, anomaly detection, status pages, and on-call alerting.",
				Subdomain:   "https://monitoring.opsway.eu",
				Tiers: []CatalogAddonTier{
					{
						Tier:         entities.EntitlementTierPro,
						Name:         "Monitoring Pro",
						Price:        9,
						Currency:     "eur",
						LookupKey:    "addon_monitoring_pro",
						MonthlyQuota: entities.DefaultMonthlyQuota(entities.ProductMonitoring, entities.EntitlementTierPro),
						QuotaUnit:    "monitors",
					},
					{
						Tier:         entities.EntitlementTierEnterprise,
						Name:         "Monitoring Enterprise",
						Price:        39,
						Currency:     "eur",
						LookupKey:    "addon_monitoring_enterprise",
						MonthlyQuota: entities.DefaultMonthlyQuota(entities.ProductMonitoring, entities.EntitlementTierEnterprise),
						QuotaUnit:    "monitors",
					},
				},
			},
			{
				Product:     entities.ProductLogSearch,
				Name:        "Opsway Log Search",
				Description: "Semantic vector & SQL log engine, OTLP ingestion, and built-in MCP server.",
				Subdomain:   "https://logs.opsway.eu",
				Tiers: []CatalogAddonTier{
					{
						Tier:         entities.EntitlementTierPro,
						Name:         "Log Search Pro",
						Price:        12,
						Currency:     "eur",
						LookupKey:    "addon_log_search_pro",
						MonthlyQuota: entities.DefaultMonthlyQuota(entities.ProductLogSearch, entities.EntitlementTierPro),
						QuotaUnit:    "GB",
					},
					{
						Tier:         entities.EntitlementTierEnterprise,
						Name:         "Log Search Enterprise",
						Price:        49,
						Currency:     "eur",
						LookupKey:    "addon_log_search_enterprise",
						MonthlyQuota: entities.DefaultMonthlyQuota(entities.ProductLogSearch, entities.EntitlementTierEnterprise),
						QuotaUnit:    "GB",
					},
				},
			},
			{
				Product:     entities.ProductPantomath,
				Name:        "Agent Context",
				Description: "Git AST parsing & Kubernetes manifest GraphRAG for line-level root-cause diagnosis.",
				Subdomain:   "https://agent-context.opsway.eu",
				Tiers: []CatalogAddonTier{
					{
						Tier:         entities.EntitlementTierPro,
						Name:         "Agent Context Pro",
						Price:        12,
						Currency:     "eur",
						LookupKey:    "addon_pantomath_pro",
						MonthlyQuota: entities.DefaultMonthlyQuota(entities.ProductPantomath, entities.EntitlementTierPro),
						QuotaUnit:    "repos",
					},
					{
						Tier:         entities.EntitlementTierEnterprise,
						Name:         "Agent Context Enterprise",
						Price:        49,
						Currency:     "eur",
						LookupKey:    "addon_pantomath_enterprise",
						MonthlyQuota: entities.DefaultMonthlyQuota(entities.ProductPantomath, entities.EntitlementTierEnterprise),
						QuotaUnit:    "repos",
					},
				},
			},
			{
				Product:     entities.ProductManagedAgent,
				Name:        "Autonomous Agents",
				Description: "Managed AI SRE investigations, Unified MCP Gateway, and automated remediation.",
				Subdomain:   "https://my.opsway.eu/agents",
				Tiers: []CatalogAddonTier{
					{
						Tier:         entities.EntitlementTierPro,
						Name:         "Autonomous Agents Pro",
						Price:        15,
						Currency:     "eur",
						LookupKey:    "addon_managed_agent_pro",
						MonthlyQuota: entities.DefaultMonthlyQuota(entities.ProductManagedAgent, entities.EntitlementTierPro),
						QuotaUnit:    "runs",
					},
					{
						Tier:         entities.EntitlementTierEnterprise,
						Name:         "Autonomous Agents Enterprise",
						Price:        59,
						Currency:     "eur",
						LookupKey:    "addon_managed_agent_enterprise",
						MonthlyQuota: entities.DefaultMonthlyQuota(entities.ProductManagedAgent, entities.EntitlementTierEnterprise),
						QuotaUnit:    "runs",
					},
				},
			},
		},
	}
}

// ValidateReturnURL ensures Stripe redirect URLs only target trusted Opsway domains
// or localhost during development, preventing open-redirect abuse.
func ValidateReturnURL(requested, defaultDomain string) string {
	fallback := strings.TrimRight(defaultDomain, "/") + "/settings/billing"
	if strings.TrimSpace(defaultDomain) == "" {
		fallback = "https://my.opsway.eu/settings/billing"
	}

	candidate := strings.TrimSpace(requested)
	if candidate == "" {
		return fallback
	}

	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fallback
	}

	host := strings.ToLower(parsed.Hostname())
	switch {
	case (parsed.Scheme == "https") && (host == "opsway.eu" || strings.HasSuffix(host, ".opsway.eu")):
		return candidate
	case (parsed.Scheme == "http" || parsed.Scheme == "https") && (host == "localhost" || host == "127.0.0.1"):
		return candidate
	default:
		return fallback
	}
}
