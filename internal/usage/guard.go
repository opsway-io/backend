package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/opsway-io/backend/internal/billing"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// Grace buffer policies: shorter for Free (24h / +10%), longer for Paid (72h / +15%).
	FreeGraceDuration = 24 * time.Hour
	FreeGraceCapPct   = 110.0

	PaidGraceDuration = 72 * time.Hour
	PaidGraceCapPct   = 115.0

	WarningThresholdPct = 80.0
)

// Unit overage rates in cents per unit (or per block for high-count metrics) on paid plans.
var overageUnitCents = map[string]int{
	"ingestion_gb":           50,   // $0.50 / GB
	"managed_investigations": 25,   // $0.25 / AI run
	"monitors":               100,  // $1.00 / monitor
	"indexed_products":       1000, // $10.00 / Pantomath product
	"indexed_repos":          200,  // $2.00 / Pantomath repo
}

func estimateOverageCents(quotaKey string, overUnits float64) int {
	if overUnits <= 0 {
		return 0
	}
	switch quotaKey {
	case "monthly_tokens":
		// $5.00 per 1M overage tokens
		return int(math.Ceil((overUnits / 1_000_000.0) * 500.0))
	case "vector_chunks":
		// $0.50 per 10k overage vector chunks
		return int(math.Ceil((overUnits / 10_000.0) * 50.0))
	default:
		if unitPrice := overageUnitCents[quotaKey]; unitPrice > 0 {
			return int(math.Ceil(overUnits * float64(unitPrice)))
		}
		return 0
	}
}

type OverageSettings struct {
	AllowOverage          bool `json:"allowOverage"`
	OverageSpendCapCents  *int `json:"overageSpendCapCents,omitempty"`
	EstimatedOverageCents int  `json:"estimatedOverageCents"`
	Eligible              bool `json:"eligible"` // true on TEAM / ENTERPRISE
}

type ProductEnforcementSummary struct {
	Product               entities.ProductSlug           `json:"product"`
	QuotaKey              string                         `json:"quotaKey"`
	Label                 string                         `json:"label"`
	State                 entities.QuotaEnforcementState `json:"state"`
	Used                  float64                        `json:"used"`
	Limit                 float64                        `json:"limit"`
	Unit                  string                         `json:"unit"`
	Percent               float64                        `json:"percent"`
	GraceExpiresAt        *time.Time                     `json:"graceExpiresAt,omitempty"`
	GraceHoursRemaining   *int                           `json:"graceHoursRemaining,omitempty"`
	EstimatedOverageCents int                            `json:"estimatedOverageCents"`
	TopConsumerHint       string                         `json:"topConsumerHint,omitempty"`
	BehaviorSummary       string                         `json:"behaviorSummary"`
}

// ComputeEnforcementTransition determines the next QuotaEnforcementState given plan, quota,
// existing grace window, and pay-as-you-go overage settings.
func ComputeEnforcementTransition(
	plan entities.PaymentPlan,
	allowOverage bool,
	spendCapCents *int,
	quotaKey string,
	used, limit float64,
	existingGraceStart *time.Time,
	now time.Time,
) (state entities.QuotaEnforcementState, graceStart *time.Time, graceEnd *time.Time, overageCents int) {
	if limit < 0 || limit == 0 {
		return entities.QuotaStateNormal, nil, nil, 0
	}

	pct := (used / limit) * 100.0
	if pct < WarningThresholdPct {
		return entities.QuotaStateNormal, nil, nil, 0
	}
	if pct < 100.0 {
		return entities.QuotaStateWarning, nil, nil, 0
	}

	// Over 100% — calculate overage units & estimated cost
	overUnits := math.Max(0, used-limit)
	overageCents = estimateOverageCents(quotaKey, overUnits)

	isPaid := plan == entities.PaymentPlanPro || plan == entities.PaymentPlanTeam || plan == entities.PaymentPlanEnterprise

	// Paid plan with Pay-As-You-Go overages enabled stays unthrottled unless spend cap is hit.
	if isPaid && allowOverage {
		if spendCapCents == nil || *spendCapCents <= 0 || overageCents < *spendCapCents {
			return entities.QuotaStateOverageBilling, nil, nil, overageCents
		}
		return entities.QuotaStateThrottled, existingGraceStart, nil, overageCents
	}

	// Grace buffer: 24h / +10% for Free; 72h / +15% for Paid
	graceDur := FreeGraceDuration
	graceCap := FreeGraceCapPct
	if isPaid {
		graceDur = PaidGraceDuration
		graceCap = PaidGraceCapPct
	}

	start := now
	if existingGraceStart != nil && !existingGraceStart.IsZero() {
		start = *existingGraceStart
	}
	end := start.Add(graceDur)

	if pct >= graceCap || !now.Before(end) {
		return entities.QuotaStateThrottled, &start, &end, overageCents
	}
	return entities.QuotaStateGrace, &start, &end, overageCents
}

func DescribeEnforcementBehavior(product entities.ProductSlug, state entities.QuotaEnforcementState) string {
	switch state {
	case entities.QuotaStateNormal:
		return "Operating normally within plan quota."
	case entities.QuotaStateWarning:
		return "Approaching monthly quota (80%+). No service impact yet."
	case entities.QuotaStateOverageBilling:
		return "Pay-as-you-go overage active — full service continues without throttling."
	case entities.QuotaStateGrace:
		switch product {
		case entities.ProductLogSearch:
			return "Grace period active: vector embeddings & DEBUG logs paused; INFO/WARN/ERROR logs still ingesting."
		case entities.ProductManagedAgent:
			return "Grace period active: manual investigations continue; automatic low-severity triggers paused."
		case entities.ProductPantomath:
			return "Grace period active: existing code graph remains fully searchable."
		default:
			return "Grace period active: existing resources continue running normally."
		}
	case entities.QuotaStateThrottled:
		switch product {
		case entities.ProductLogSearch:
			return "Throttled (zero data loss): existing logs stay searchable, ERROR logs pass through, excess INFO/WARN logs spill to cold storage."
		case entities.ProductManagedAgent:
			return "Throttled: incidents fall back to BYO-LLM/BYOA or deterministic log summary triage."
		case entities.ProductPantomath:
			return "Only 1 product remains searchable; excess products are locked and scheduled for deletion in 7 days unless you upgrade."
		default:
			return "Quota reached: existing data preserved; new resource creation paused until upgrade or reset."
		}
	default:
		return ""
	}
}

type quotaAlertTemplate struct {
	subject string
	plain   string
	html    string
}

func (t quotaAlertTemplate) Subject() string   { return t.subject }
func (t quotaAlertTemplate) PlainText() string { return t.plain }
func (t quotaAlertTemplate) HTML() string      { return t.html }

// Guard periodically evaluates team usage, updates QuotaNotificationState, and sends
// deduplicated Email + Team Webhook alerts when thresholds are crossed.
type Guard struct {
	db          *gorm.DB
	usageSvc    Service
	billingSvc  billing.Service
	emailSender email.Sender
	appDomain   string
	httpClient  *http.Client
	logger      *logrus.Entry
}

func NewGuard(
	db *gorm.DB,
	usageSvc Service,
	billingSvc billing.Service,
	emailSender email.Sender,
	appDomain string,
	logger *logrus.Entry,
) *Guard {
	if appDomain == "" {
		appDomain = "https://my.opsway.eu"
	}
	return &Guard{
		db:          db,
		usageSvc:    usageSvc,
		billingSvc:  billingSvc,
		emailSender: emailSender,
		appDomain:   strings.TrimRight(appDomain, "/"),
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		logger:      logger,
	}
}

func (g *Guard) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		defer ticker.Stop()
		// Initial delayed pass 30s after boot so services finish warming up.
		select {
		case <-time.After(30 * time.Second):
			g.EvaluateAllTeams(ctx)
		case <-ctx.Done():
			return
		}
		for {
			select {
			case <-ticker.C:
				g.EvaluateAllTeams(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (g *Guard) EvaluateAllTeams(ctx context.Context) {
	var teams []entities.Team
	if err := g.db.WithContext(ctx).Find(&teams).Error; err != nil {
		if g.logger != nil {
			g.logger.WithError(err).Warn("usage-guard: failed to list teams")
		}
		return
	}
	for i := range teams {
		g.ReconcileTeamStripePlan(ctx, &teams[i])
		if _, err := g.EvaluateTeam(ctx, teams[i].ID); err != nil && g.logger != nil {
			g.logger.WithError(err).WithField("teamId", teams[i].ID).Warn("usage-guard: failed to evaluate team")
		}
	}
}

func (g *Guard) EvaluateTeam(ctx context.Context, teamID uint) ([]ProductEnforcementSummary, error) {
	now := time.Now().UTC()
	var team entities.Team
	if err := g.db.WithContext(ctx).First(&team, teamID).Error; err != nil {
		return nil, err
	}

	usageResp, err := g.usageSvc.GetTeamUsage(ctx, teamID, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	cycleStart := usageResp.BillingCycle.Start.Format("2006-01-02")

	type evalTarget struct {
		product entities.ProductSlug
		quota   QuotaItem
		topHint string
	}

	topLogHint := ""
	if len(usageResp.TopConsumers.LogServices) > 0 {
		top := usageResp.TopConsumers.LogServices[0]
		topLogHint = fmt.Sprintf("Top log service: %s (%.1f%% of volume)", top.ServiceName, top.SharePct)
	}
	topMonHint := ""
	if len(usageResp.TopConsumers.Monitors) > 0 {
		top := usageResp.TopConsumers.Monitors[0]
		topMonHint = fmt.Sprintf("Most active monitor: %s (%d probes)", top.Name, top.Probes)
	}
	topAgentHint := ""
	if len(usageResp.TopConsumers.AgentTriggers) > 0 {
		top := usageResp.TopConsumers.AgentTriggers[0]
		topAgentHint = fmt.Sprintf("Top trigger source: %s (%d runs)", top.TriggerSource, top.Runs)
	}

	var targets []evalTarget
	for _, q := range usageResp.Products.Monitoring.Quotas {
		if q.Key == "monitors" {
			targets = append(targets, evalTarget{entities.ProductMonitoring, q, topMonHint})
		}
	}
	for _, q := range usageResp.Products.LogSearch.Quotas {
		targets = append(targets, evalTarget{entities.ProductLogSearch, q, topLogHint})
	}
	for _, q := range usageResp.Products.ManagedAgent.Quotas {
		targets = append(targets, evalTarget{entities.ProductManagedAgent, q, topAgentHint})
	}
	for _, q := range usageResp.Products.Pantomath.Quotas {
		targets = append(targets, evalTarget{entities.ProductPantomath, q, ""})
	}

	summaries := make([]ProductEnforcementSummary, 0, len(targets))

	for _, t := range targets {
		var existing entities.QuotaNotificationState
		err := g.db.WithContext(ctx).
			Where("team_id = ? AND product = ? AND quota_key = ? AND billing_cycle_start = ?",
				teamID, t.product, t.quota.Key, cycleStart).
			First(&existing).Error
		hasExisting := err == nil

		var prevGraceStart *time.Time
		var prevState entities.QuotaEnforcementState
		if hasExisting {
			prevGraceStart = existing.GraceStartedAt
			prevState = existing.State
		}

		nextState, graceStart, graceEnd, overageCents := ComputeEnforcementTransition(
			team.PaymentPlan,
			team.AllowOverage,
			team.OverageSpendCapCents,
			t.quota.Key,
			t.quota.Used,
			t.quota.Limit,
			prevGraceStart,
			now,
		)

		row := existing
		row.TeamID = teamID
		row.Product = t.product
		row.QuotaKey = t.quota.Key
		row.BillingCycleStart = cycleStart
		row.State = nextState
		row.UsedValue = t.quota.Used
		row.LimitValue = t.quota.Limit
		row.EstimatedOverageCents = overageCents
		row.TopConsumerHint = t.topHint
		row.GraceStartedAt = graceStart
		row.GraceExpiresAt = graceEnd

		shouldNotify := false
		switch nextState {
		case entities.QuotaStateWarning:
			if row.NotifiedWarningAt == nil {
				row.NotifiedWarningAt = &now
				shouldNotify = true
			}
		case entities.QuotaStateGrace:
			if row.NotifiedGraceAt == nil {
				row.NotifiedGraceAt = &now
				shouldNotify = true
			}
		case entities.QuotaStateOverageBilling:
			if row.NotifiedOverageAt == nil {
				row.NotifiedOverageAt = &now
				shouldNotify = true
			}
		case entities.QuotaStateThrottled:
			if row.NotifiedThrottledAt == nil {
				row.NotifiedThrottledAt = &now
				shouldNotify = true
			}
		case entities.QuotaStateNormal:
			if hasExisting && (prevState == entities.QuotaStateGrace || prevState == entities.QuotaStateThrottled) && row.NotifiedRecoveredAt == nil {
				row.NotifiedRecoveredAt = &now
				shouldNotify = true
			}
		}

		if err := g.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "team_id"},
				{Name: "product"},
				{Name: "quota_key"},
				{Name: "billing_cycle_start"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"state", "used_value", "limit_value", "estimated_overage_cents",
				"top_consumer_hint", "grace_started_at", "grace_expires_at",
				"notified_warning_at", "notified_grace_at", "notified_overage_at",
				"notified_throttled_at", "notified_recovered_at", "updated_at",
			}),
		}).Create(&row).Error; err != nil && g.logger != nil {
			g.logger.WithError(err).Warn("usage-guard: failed to upsert quota state")
		}

		if shouldNotify {
			g.dispatchUnifiedNotification(ctx, &team, t.product, t.quota, nextState, graceEnd, t.topHint)
		}

		var graceHours *int
		if graceEnd != nil && nextState == entities.QuotaStateGrace {
			hrs := int(math.Ceil(time.Until(*graceEnd).Hours()))
			if hrs < 0 {
				hrs = 0
			}
			graceHours = &hrs
		}

		summaries = append(summaries, ProductEnforcementSummary{
			Product:               t.product,
			QuotaKey:              t.quota.Key,
			Label:                 t.quota.Label,
			State:                 nextState,
			Used:                  t.quota.Used,
			Limit:                 t.quota.Limit,
			Unit:                  t.quota.Unit,
			Percent:               t.quota.Percent,
			GraceExpiresAt:        graceEnd,
			GraceHoursRemaining:   graceHours,
			EstimatedOverageCents: overageCents,
			TopConsumerHint:       t.topHint,
			BehaviorSummary:       DescribeEnforcementBehavior(t.product, nextState),
		})
	}

	return summaries, nil
}

func (g *Guard) GetProductEnforcementState(ctx context.Context, teamID uint, product entities.ProductSlug) entities.QuotaEnforcementState {
	cycleStart := calculateBillingCycle(time.Now().UTC()).Start.Format("2006-01-02")
	var rows []entities.QuotaNotificationState
	if err := g.db.WithContext(ctx).
		Where("team_id = ? AND product = ? AND billing_cycle_start = ?", teamID, product, cycleStart).
		Find(&rows).Error; err != nil || len(rows) == 0 {
		return entities.QuotaStateNormal
	}
	// Return the strictest active enforcement state across all quotas for this product.
	rank := func(s entities.QuotaEnforcementState) int {
		switch s {
		case entities.QuotaStateThrottled:
			return 4
		case entities.QuotaStateGrace:
			return 3
		case entities.QuotaStateOverageBilling:
			return 2
		case entities.QuotaStateWarning:
			return 1
		default:
			return 0
		}
	}
	worst := entities.QuotaStateNormal
	for _, r := range rows {
		if rank(r.State) > rank(worst) {
			worst = r.State
		}
	}
	return worst
}

func (g *Guard) GetTeamMCPRateLimit(ctx context.Context, teamID uint) int {
	var team entities.Team
	if err := g.db.WithContext(ctx).Select("id, payment_plan").First(&team, teamID).Error; err != nil {
		return 60
	}
	return team.GetMCPGatewayRateLimitPerMin()
}

func (g *Guard) GetTeamTriggerRuleLimit(ctx context.Context, teamID uint) int {
	var team entities.Team
	if err := g.db.WithContext(ctx).Select("id, payment_plan").First(&team, teamID).Error; err != nil {
		return 3
	}
	return team.GetAgentTriggerRuleLimit()
}

func (g *Guard) GetTeamPlanLimits(ctx context.Context, teamID uint) map[string]interface{} {
	var team entities.Team
	if err := g.db.WithContext(ctx).Select("id, payment_plan, allow_overage").First(&team, teamID).Error; err != nil {
		team.PaymentPlan = entities.PaymentPlanFree
	}
	return map[string]interface{}{
		"plan":                team.PaymentPlan,
		"allowOverage":        team.AllowOverage,
		"maxEntitlementTier":  team.MaxEntitlementTier(),
		"maxRepos":            team.GetPantomathRepoLimit(),
		"maxVectorChunks":     team.GetPantomathVectorChunkLimit(),
		"maxTokensPerRun":     team.GetAgentTokenLimitPerRun(),
		"maxMonthlyTokens":    team.GetAgentMonthlyTokenLimit(),
		"mcpRateLimitPerMin":  team.GetMCPGatewayRateLimitPerMin(),
		"minMonitorFreqSecs":  team.GetMinMonitorFrequencySeconds(),
		"maxMonitorSteps":     team.GetMaxMonitorSteps(),
	}
}

func (g *Guard) dispatchUnifiedNotification(
	ctx context.Context,
	team *entities.Team,
	product entities.ProductSlug,
	quota QuotaItem,
	state entities.QuotaEnforcementState,
	graceEnd *time.Time,
	topHint string,
) {
	behavior := DescribeEnforcementBehavior(product, state)
	usageURL := fmt.Sprintf("%s/settings/usage", g.appDomain)
	billingURL := fmt.Sprintf("%s/settings/billing", g.appDomain)

	var title string
	switch state {
	case entities.QuotaStateWarning:
		title = fmt.Sprintf("[Opsway] %s reached %.0f%% of monthly quota", quota.Label, quota.Percent)
	case entities.QuotaStateGrace:
		graceNote := "Grace period active"
		if graceEnd != nil {
			graceNote = fmt.Sprintf("Grace expires %s UTC", graceEnd.UTC().Format("Jan 2 15:04"))
		}
		title = fmt.Sprintf("[Opsway] %s exceeded 100%% quota (%s)", quota.Label, graceNote)
	case entities.QuotaStateOverageBilling:
		title = fmt.Sprintf("[Opsway] %s entered Pay-As-You-Go overage", quota.Label)
	case entities.QuotaStateThrottled:
		title = fmt.Sprintf("[Opsway] %s quota exceeded — protection mode active", quota.Label)
	case entities.QuotaStateNormal:
		title = fmt.Sprintf("[Opsway] %s quota restored to normal", quota.Label)
	}

	plain := fmt.Sprintf(
		"%s\n\nTeam: %s (%s plan)\nUsage: %.2f / %.2f %s (%.0f%%)\n%s\n\nWhat happens now:\n%s\n\nView usage: %s\nManage plan & overages: %s\n",
		title, team.Name, team.PaymentPlan, quota.Used, quota.Limit, quota.Unit, quota.Percent,
		topHint, behavior, usageURL, billingURL,
	)
	html := fmt.Sprintf(
		"<h3>%s</h3><p><strong>Team:</strong> %s (%s plan)<br/><strong>Usage:</strong> %.2f / %.2f %s (%.0f%%)</p><p>%s</p><p><strong>What happens now:</strong> %s</p><p><a href=\"%s\">View Usage Breakdown</a> · <a href=\"%s\">Manage Plan &amp; Overages</a></p>",
		title, team.Name, team.PaymentPlan, quota.Used, quota.Limit, quota.Unit, quota.Percent,
		topHint, behavior, usageURL, billingURL,
	)

	// 1. Email team OWNER & ADMIN users
	if g.emailSender != nil {
		type adminRecipient struct {
			Name  string
			Email string
		}
		var admins []adminRecipient
		g.db.WithContext(ctx).Table("users").
			Select("users.name, users.email").
			Joins("JOIN team_users ON team_users.user_id = users.id").
			Where("team_users.team_id = ? AND team_users.role IN ?", team.ID, []entities.TeamRole{entities.TeamRoleOwner, entities.TeamRoleAdmin}).
			Scan(&admins)
		tmpl := quotaAlertTemplate{subject: title, plain: plain, html: html}
		for _, a := range admins {
			if a.Email == "" {
				continue
			}
			if err := g.emailSender.Send(ctx, a.Name, a.Email, tmpl); err != nil && g.logger != nil {
				g.logger.WithError(err).WithField("email", a.Email).Warn("usage-guard: failed to send quota email")
			}
		}
	}

	// 2. Team chat webhooks (Slack, Discord, MS Teams, generic webhook)
	webhookText := fmt.Sprintf("*%s*\nUsage: %.2f / %.2f %s (%.0f%%)\n%s\n%s\n<%s|View Usage> | <%s|Manage Plan>",
		title, quota.Used, quota.Limit, quota.Unit, quota.Percent, topHint, behavior, usageURL, billingURL)

	postJSON := func(url string, payload interface{}) {
		if url == "" {
			return
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := g.httpClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}

	if team.SlackWebhookURL != nil && *team.SlackWebhookURL != "" {
		postJSON(*team.SlackWebhookURL, map[string]string{"text": webhookText})
	}
	if team.DiscordWebhookURL != nil && *team.DiscordWebhookURL != "" {
		postJSON(*team.DiscordWebhookURL, map[string]string{"content": webhookText})
	}
	if team.MicrosoftTeamsWebhookURL != nil && *team.MicrosoftTeamsWebhookURL != "" {
		postJSON(*team.MicrosoftTeamsWebhookURL, map[string]string{"text": webhookText})
	}
	if team.WebhookURL != nil && *team.WebhookURL != "" {
		postJSON(*team.WebhookURL, map[string]interface{}{
			"event":       "quota.state_changed",
			"teamId":      team.ID,
			"product":     product,
			"quotaKey":    quota.Key,
			"state":       state,
			"used":        quota.Used,
			"limit":       quota.Limit,
			"unit":        quota.Unit,
			"percent":     quota.Percent,
			"topConsumer": topHint,
			"behavior":    behavior,
			"usageUrl":    usageURL,
			"billingUrl":  billingURL,
		})
	}
}
