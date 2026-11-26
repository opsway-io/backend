package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/investigation"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BillingCycleInfo struct {
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	DaysElapsed   int       `json:"daysElapsed"`
	DaysTotal     int       `json:"daysTotal"`
	DaysRemaining int       `json:"daysRemaining"`
}

type QuotaItem struct {
	Key        string   `json:"key"`
	Label      string   `json:"label"`
	Used       float64  `json:"used"`
	Limit      float64  `json:"limit"` // -1 means unlimited
	Unit       string   `json:"unit"`
	Percent    float64  `json:"percent"` // 0..100+, 0 when unlimited
	Projected  *float64 `json:"projected,omitempty"`
	Status     string   `json:"status"` // "ok", "warning", "critical", "unlimited"
}

type MonitoringUsage struct {
	Tier                  entities.EntitlementTier `json:"tier"`
	Enabled               bool                     `json:"enabled"`
	Quotas                []QuotaItem              `json:"quotas"`
	ProbeExecutions       int64                    `json:"probeExecutions"`
	ActiveHeartbeats      int64                    `json:"activeHeartbeats"`
	NotificationsSent     int64                    `json:"notificationsSent"`
	IncidentsOpened       int64                    `json:"incidentsOpened"`
	StatusPageSubscribers int64                    `json:"statusPageSubscribers"`
}

type LogSearchUsage struct {
	Tier                       entities.EntitlementTier `json:"tier"`
	Enabled                    bool                     `json:"enabled"`
	Quotas                     []QuotaItem              `json:"quotas"`
	EventsIngested             int64                    `json:"eventsIngested"`
	BytesIngested              int64                    `json:"bytesIngested"`
	ActiveServices             int64                    `json:"activeServices"`
	TriggerRules               int64                    `json:"triggerRules"`
	LogTriggeredInvestigations int64                    `json:"logTriggeredInvestigations"`
}

type ManagedAgentUsage struct {
	Tier                  entities.EntitlementTier `json:"tier"`
	Enabled               bool                     `json:"enabled"`
	Quotas                []QuotaItem              `json:"quotas"`
	ManagedRuns           int64                    `json:"managedRuns"`
	ByoaRuns              int64                    `json:"byoaRuns"`
	TokensUsed            int64                    `json:"tokensUsed"`
	AvgDurationMs         int64                    `json:"avgDurationMs"`
	MCPCalls              int64                    `json:"mcpCalls"`
	RemediationsSuggested int64                    `json:"remediationsSuggested"`
	RemediationsExecuted  int64                    `json:"remediationsExecuted"`
	ActiveTriggerRules    int64                    `json:"activeTriggerRules"`
}

type PantomathUsage struct {
	Tier                             entities.EntitlementTier `json:"tier"`
	Enabled                          bool                     `json:"enabled"`
	Quotas                           []QuotaItem              `json:"quotas"`
	LinkedRepositories               int64                    `json:"linkedRepositories"`
	VectorChunks                     int64                    `json:"vectorChunks"`
	GraphEdges                       int64                    `json:"graphEdges"`
	QueriesExecuted                  int64                    `json:"queriesExecuted"`
	PantomathTriggeredInvestigations int64                    `json:"pantomathTriggeredInvestigations"`
	MCPCalls                         int64                    `json:"mcpCalls"`
}

type ProductsUsage struct {
	Monitoring   MonitoringUsage   `json:"monitoring"`
	LogSearch    LogSearchUsage    `json:"logSearch"`
	ManagedAgent ManagedAgentUsage `json:"managedAgent"`
	Pantomath    PantomathUsage    `json:"pantomath"`
}

type PlatformUsage struct {
	TeamSeats           QuotaItem `json:"teamSeats"`
	ActiveMembers       int64     `json:"activeMembers"`
	PendingInvitations  int64     `json:"pendingInvitations"`
	ServiceCatalogCount int64     `json:"serviceCatalogCount"`
	ActiveAPIKeys       int64     `json:"activeApiKeys"`
	ReportsGenerated    int64     `json:"reportsGenerated"`
}

type DailyUsagePoint struct {
	Date          string `json:"date"` // YYYY-MM-DD
	Probes        int64  `json:"probes"`
	LogBytes      int64  `json:"logBytes"`
	LogEvents     int64  `json:"logEvents"`
	AgentRuns     int64  `json:"agentRuns"`
	AgentTokens   int64  `json:"agentTokens"`
	MCPCalls      int64  `json:"mcpCalls"`
	Notifications int64  `json:"notifications"`
	Incidents     int64  `json:"incidents"`
}

type TopLogServiceConsumer struct {
	ServiceName string  `json:"serviceName"`
	Events      int64   `json:"events"`
	Bytes       int64   `json:"bytes"`
	SharePct    float64 `json:"sharePct"`
}

type TopMonitorConsumer struct {
	MonitorID uint64  `json:"monitorId"`
	Name      string  `json:"name"`
	Probes    int64   `json:"probes"`
	SharePct  float64 `json:"sharePct"`
}

type TopTriggerConsumer struct {
	TriggerSource string `json:"triggerSource"`
	Runs          int64  `json:"runs"`
	TokensUsed    int64  `json:"tokensUsed"`
	AvgDurationMs int64  `json:"avgDurationMs"`
}

type TopServiceConsumer struct {
	ServiceName string `json:"serviceName"`
	Runs        int64  `json:"runs"`
	TokensUsed  int64  `json:"tokensUsed"`
}

type TopPantomathProductConsumer struct {
	Product      string     `json:"product"`
	RepoCount    int64      `json:"repoCount"`
	VectorChunks int64      `json:"vectorChunks"`
	Source       string     `json:"source"`
	Indexing     bool       `json:"indexing,omitempty"`
	Status       string     `json:"status"` // "active" | "scheduled_delete" | "deleted"
	DeleteAt     *time.Time `json:"deleteAt,omitempty"`
}

type TopConsumers struct {
	Monitors          []TopMonitorConsumer          `json:"monitors"`
	LogServices       []TopLogServiceConsumer       `json:"logServices"`
	PantomathProducts []TopPantomathProductConsumer `json:"pantomathProducts"`
	AgentTriggers     []TopTriggerConsumer          `json:"agentTriggers"`
	Services          []TopServiceConsumer          `json:"services"`
}

type TeamUsageResponse struct {
	TeamID       uint                        `json:"teamId"`
	Plan         string                      `json:"plan"`
	RangeStart   time.Time                   `json:"rangeStart"`
	RangeEnd     time.Time                   `json:"rangeEnd"`
	BillingCycle BillingCycleInfo            `json:"billingCycle"`
	Overage      OverageSettings             `json:"overage"`
	Enforcement  []ProductEnforcementSummary `json:"enforcement"`
	Products     ProductsUsage               `json:"products"`
	Platform     PlatformUsage               `json:"platform"`
	Timeseries   []DailyUsagePoint           `json:"timeseries"`
	TopConsumers TopConsumers                `json:"topConsumers"`
}

type Service interface {
	GetTeamUsage(ctx context.Context, teamID uint, from, to time.Time) (*TeamUsageResponse, error)
	IncrementCounter(ctx context.Context, teamID uint, product entities.ProductSlug, metricKey string, delta int64) error
	UpdateOverageSettings(ctx context.Context, teamID uint, allowOverage bool, spendCapCents *int) (*OverageSettings, error)
}

type serviceImpl struct {
	db                   *gorm.DB
	ch                   *gorm.DB
	investigationService investigation.Service
	logSearchURL         string
	logSearchAgentKey    string
	pantomathURL         string
	httpClient           *http.Client
	logger               *logrus.Entry
}

func NewService(
	db *gorm.DB,
	ch *gorm.DB,
	investigationService investigation.Service,
	logSearchURL string,
	logSearchAgentKey string,
	pantomathURL string,
	logger *logrus.Entry,
) Service {
	return &serviceImpl{
		db:                   db,
		ch:                   ch,
		investigationService: investigationService,
		logSearchURL:         strings.TrimRight(logSearchURL, "/"),
		logSearchAgentKey:    logSearchAgentKey,
		pantomathURL:         strings.TrimRight(pantomathURL, "/"),
		httpClient:           &http.Client{Timeout: 5 * time.Second},
		logger:               logger,
	}
}

type pantomathUsageAPIResp struct {
	Tenant          string `json:"tenant"`
	IndexedProducts int64  `json:"indexed_products"`
	IndexedRepos    int64  `json:"indexed_repos"`
	VectorChunks    int64  `json:"vector_chunks"`
	GraphEdges      int64  `json:"graph_edges"`
	QueriesExecuted int64  `json:"queries_executed"`
	Products        []struct {
		Product      string `json:"product"`
		RepoCount    int64  `json:"repo_count"`
		VectorChunks int64  `json:"vector_chunks"`
		Source       string `json:"source"`
		Indexing     bool   `json:"indexing"`
	} `json:"products"`
}

func (s *serviceImpl) fetchPantomathUsageForTenant(ctx context.Context, tenant string) (*pantomathUsageAPIResp, error) {
	if s.pantomathURL == "" {
		return nil, nil
	}
	u := fmt.Sprintf("%s/v1/usage", s.pantomathURL)
	if tenant != "" {
		u = fmt.Sprintf("%s?tenant=%s", u, tenant)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if tenant != "" {
		req.Header.Set("X-Opsway-Tenant-ID", tenant)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pantomath /v1/usage returned status %d", resp.StatusCode)
	}
	var parsed pantomathUsageAPIResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (s *serviceImpl) fetchPantomathUsage(ctx context.Context, teamID uint) *pantomathUsageAPIResp {
	if s.pantomathURL == "" {
		return nil
	}
	teamTenant := fmt.Sprintf("team_%d", teamID)
	res, err := s.fetchPantomathUsageForTenant(ctx, teamTenant)
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).Warn("failed to fetch pantomath usage")
		}
		return nil
	}
	if res != nil && res.IndexedProducts == 0 && res.VectorChunks == 0 {
		if fallback, err := s.fetchPantomathUsageForTenant(ctx, ""); err == nil && fallback != nil && (fallback.IndexedProducts > 0 || fallback.VectorChunks > 0) {
			return fallback
		}
	}
	return res
}

type logSearchUsageAPIResp struct {
	TenantID       string  `json:"tenant_id"`
	EventsIngested int64   `json:"events_ingested"`
	BytesIngested  int64   `json:"bytes_ingested"`
	IngestionGB    float64 `json:"ingestion_gb"`
	ActiveServices int64   `json:"active_services"`
	TopServices    []struct {
		ServiceName string  `json:"service_name"`
		Events      int64   `json:"events"`
		Bytes       int64   `json:"bytes"`
		SharePct    float64 `json:"share_pct"`
	} `json:"top_services"`
	Daily []struct {
		Date   string `json:"date"`
		Events int64  `json:"events"`
		Bytes  int64  `json:"bytes"`
	} `json:"daily"`
}

func (s *serviceImpl) fetchLogSearchUsageForTenant(
	ctx context.Context,
	tenantHeader string,
	from, to time.Time,
) (*logSearchUsageAPIResp, error) {
	if s.logSearchURL == "" {
		return nil, nil
	}
	u := fmt.Sprintf("%s/v1/usage?start_ts=%d&end_ts=%d", s.logSearchURL, from.UnixMilli(), to.UnixMilli())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	apiKey := s.logSearchAgentKey
	if apiKey == "" {
		apiKey = "valid_agent_key"
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	if tenantHeader != "" {
		req.Header.Set("X-Opsway-Tenant-ID", tenantHeader)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("log-search /v1/usage returned status %d", resp.StatusCode)
	}
	var parsed logSearchUsageAPIResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (s *serviceImpl) fetchLogSearchUsage(ctx context.Context, teamID uint, from, to time.Time) *logSearchUsageAPIResp {
	if s.logSearchURL == "" {
		return nil
	}
	teamTenant := fmt.Sprintf("team_%d", teamID)
	res, err := s.fetchLogSearchUsageForTenant(ctx, teamTenant, from, to)
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).Warn("failed to fetch log-search usage")
		}
		return nil
	}
	// If no team-scoped logs were found, also check the default cluster tenant (e.g. Vector default tenant_123).
	if res != nil && res.EventsIngested == 0 {
		if fallback, err := s.fetchLogSearchUsageForTenant(ctx, "", from, to); err == nil && fallback != nil && fallback.EventsIngested > 0 {
			return fallback
		}
	}
	return res
}

func (s *serviceImpl) IncrementCounter(
	ctx context.Context,
	teamID uint,
	product entities.ProductSlug,
	metricKey string,
	delta int64,
) error {
	if s.db == nil || teamID == 0 || delta == 0 {
		return nil
	}
	today := time.Now().UTC().Format("2006-01-02")
	row := entities.UsageDailyCounter{
		TeamID:    teamID,
		Date:      today,
		Product:   product,
		MetricKey: metricKey,
		Count:     delta,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "team_id"},
			{Name: "date"},
			{Name: "product"},
			{Name: "metric_key"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"count":      gorm.Expr("usage_daily_counters.count + ?", delta),
			"updated_at": time.Now().UTC(),
		}),
	}).Create(&row).Error
}

func buildQuotaItem(key, label string, used, limit float64, unit string, projected *float64) QuotaItem {
	roundedUsed := math.Round(used*100) / 100
	if used > 0 && used < 1 {
		roundedUsed = math.Round(used*10000) / 10000
	}
	item := QuotaItem{
		Key:       key,
		Label:     label,
		Used:      roundedUsed,
		Limit:     limit,
		Unit:      unit,
		Projected: projected,
	}
	if limit < 0 {
		item.Status = "unlimited"
		item.Percent = 0
		return item
	}
	if limit == 0 {
		if used > 0 {
			item.Percent = 100
			item.Status = "critical"
		} else {
			item.Percent = 0
			item.Status = "ok"
		}
		return item
	}
	pct := (used / limit) * 100
	item.Percent = math.Round(pct*10) / 10
	switch {
	case pct >= 95:
		item.Status = "critical"
	case pct >= 80:
		item.Status = "warning"
	default:
		item.Status = "ok"
	}
	return item
}

func calculateBillingCycle(now time.Time) BillingCycleInfo {
	utc := now.UTC()
	start := time.Date(utc.Year(), utc.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	daysTotal := int(end.Sub(start).Hours() / 24)
	daysElapsed := int(utc.Sub(start).Hours()/24) + 1
	if daysElapsed < 1 {
		daysElapsed = 1
	}
	if daysElapsed > daysTotal {
		daysElapsed = daysTotal
	}
	daysRemaining := daysTotal - daysElapsed
	if daysRemaining < 0 {
		daysRemaining = 0
	}
	return BillingCycleInfo{
		Start:         start,
		End:           end,
		DaysElapsed:   daysElapsed,
		DaysTotal:     daysTotal,
		DaysRemaining: daysRemaining,
	}
}

func projectMonthlyUsage(used float64, cycle BillingCycleInfo) *float64 {
	if used <= 0 || cycle.DaysElapsed <= 0 || cycle.DaysTotal <= 0 {
		return nil
	}
	projected := (used / float64(cycle.DaysElapsed)) * float64(cycle.DaysTotal)
	rounded := math.Round(projected*10) / 10
	return &rounded
}

func escalationPolicyLimit(plan entities.PaymentPlan) int {
	switch plan {
	case entities.PaymentPlanEnterprise:
		return -1
	case entities.PaymentPlanPro, entities.PaymentPlanTeam:
		return 5
	default:
		return 1
	}
}

func buildDateSeries(from, to time.Time) ([]string, map[string]*DailyUsagePoint) {
	startDay := time.Date(from.UTC().Year(), from.UTC().Month(), from.UTC().Day(), 0, 0, 0, 0, time.UTC)
	endDay := time.Date(to.UTC().Year(), to.UTC().Month(), to.UTC().Day(), 0, 0, 0, 0, time.UTC)
	if endDay.Before(startDay) {
		endDay = startDay
	}

	// Cap at 180 days to prevent unbounded responses
	if endDay.Sub(startDay).Hours()/24 > 180 {
		startDay = endDay.AddDate(0, 0, -180)
	}

	var dates []string
	byDate := make(map[string]*DailyUsagePoint)
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		dates = append(dates, key)
		byDate[key] = &DailyUsagePoint{Date: key}
	}
	return dates, byDate
}

func (s *serviceImpl) GetTeamUsage(ctx context.Context, teamID uint, from, to time.Time) (*TeamUsageResponse, error) {
	now := time.Now().UTC()
	cycle := calculateBillingCycle(now)

	if from.IsZero() {
		from = cycle.Start
	}
	if to.IsZero() {
		to = now
	}
	if to.Before(from) {
		from, to = to, from
	}

	// 1. Team & entitlements
	var team entities.Team
	if err := s.db.WithContext(ctx).First(&team, teamID).Error; err != nil {
		return nil, err
	}

	isAdminTeam := isPlatformAdminTeam(ctx, s.db, teamID)
	if isAdminTeam {
		team.PaymentPlan = entities.PaymentPlanEnterprise
	}

	entMap := make(map[entities.ProductSlug]entities.TeamEntitlement)
	if s.investigationService != nil {
		if ents, err := s.investigationService.GetEntitlements(ctx, teamID); err == nil {
			for _, e := range ents {
				entMap[e.Product] = e
			}
		}
	}

	getEnt := func(p entities.ProductSlug) entities.TeamEntitlement {
		if isAdminTeam {
			return entities.TeamEntitlement{
				TeamID:       teamID,
				Product:      p,
				Tier:         entities.EntitlementTierEnterprise,
				Enabled:      true,
				MonthlyQuota: entities.DefaultMonthlyQuota(p, entities.EntitlementTierEnterprise),
			}
		}
		if e, ok := entMap[p]; ok {
			return e
		}
		tier := entities.EntitlementTierFree
		switch team.PaymentPlan {
		case entities.PaymentPlanPro, entities.PaymentPlanTeam:
			tier = entities.EntitlementTierPro
		case entities.PaymentPlanEnterprise:
			tier = entities.EntitlementTierEnterprise
		}
		return entities.TeamEntitlement{
			TeamID:       teamID,
			Product:      p,
			Tier:         tier,
			Enabled:      true,
			MonthlyQuota: entities.DefaultMonthlyQuota(p, tier),
		}
	}

	monEnt := getEnt(entities.ProductMonitoring)
	logEnt := getEnt(entities.ProductLogSearch)
	agentEnt := getEnt(entities.ProductManagedAgent)
	pantoEnt := getEnt(entities.ProductPantomath)

	// 2. Monitoring counts (Postgres)
	var monitorCount, statusPageCount, escalationPolicyCount, heartbeatCount int64
	var incidentsOpened, notificationsSent, statusPageSubscribers int64

	s.db.WithContext(ctx).Model(&entities.Monitor{}).Where("team_id = ?", teamID).Count(&monitorCount)
	s.db.WithContext(ctx).Model(&entities.StatusPage{}).Where("team_id = ?", teamID).Count(&statusPageCount)
	s.db.WithContext(ctx).Model(&entities.EscalationPolicy{}).Where("team_id = ?", teamID).Count(&escalationPolicyCount)
	s.db.WithContext(ctx).Model(&entities.Heartbeat{}).Where("team_id = ?", teamID).Count(&heartbeatCount)
	s.db.WithContext(ctx).Model(&entities.Incident{}).
		Where("team_id = ? AND created_at >= ? AND created_at <= ?", teamID, from, to).
		Count(&incidentsOpened)
	s.db.WithContext(ctx).Table("notification_jobs").
		Joins("JOIN incidents ON incidents.id = notification_jobs.incident_id").
		Where("incidents.team_id = ? AND notification_jobs.created_at >= ? AND notification_jobs.created_at <= ?", teamID, from, to).
		Count(&notificationsSent)
	s.db.WithContext(ctx).Table("status_page_subscribers").
		Joins("JOIN status_pages ON status_pages.id = status_page_subscribers.status_page_id").
		Where("status_pages.team_id = ?", teamID).
		Count(&statusPageSubscribers)

	// 3. Platform & Governance counts (Postgres)
	var activeMembers, pendingInvitations, serviceCatalogCount, activeAPIKeys, reportsGenerated int64
	s.db.WithContext(ctx).Model(&entities.TeamUser{}).Where("team_id = ?", teamID).Count(&activeMembers)
	s.db.WithContext(ctx).Model(&entities.TeamInvitation{}).Where("team_id = ?", teamID).Count(&pendingInvitations)
	s.db.WithContext(ctx).Model(&entities.ServiceCatalogItem{}).Where("team_id = ?", teamID).Count(&serviceCatalogCount)
	s.db.WithContext(ctx).Model(&entities.APIKey{}).Where("team_id = ?", teamID).Count(&activeAPIKeys)
	s.db.WithContext(ctx).Model(&entities.Report{}).
		Where("team_id = ? AND created_at >= ? AND created_at <= ?", teamID, from, to).
		Count(&reportsGenerated)

	// 4. Autonomous Agents & Investigations counts (Postgres)
	var managedRuns, byoaRuns, activeTriggerRules, logTriggerRules int64
	var logTriggeredInv, pantoTriggeredInv int64
	var remediationsSuggested, remediationsExecuted int64

	s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
		Where("team_id = ? AND execution_mode = ? AND created_at >= ? AND created_at <= ?",
			teamID, entities.AgentModeManaged, from, to).
		Count(&managedRuns)
	s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
		Where("team_id = ? AND execution_mode != ? AND created_at >= ? AND created_at <= ?",
			teamID, entities.AgentModeManaged, from, to).
		Count(&byoaRuns)
	s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
		Where("team_id = ? AND source_product = ? AND created_at >= ? AND created_at <= ?",
			teamID, "log_search", from, to).
		Count(&logTriggeredInv)
	s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
		Where("team_id = ? AND source_product = ? AND created_at >= ? AND created_at <= ?",
			teamID, "pantomath", from, to).
		Count(&pantoTriggeredInv)

	type invAgg struct {
		TotalTokens int64
		AvgDuration float64
	}
	var agg invAgg
	s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
		Select("COALESCE(SUM(tokens_used), 0) AS total_tokens, COALESCE(AVG(duration_ms), 0) AS avg_duration").
		Where("team_id = ? AND created_at >= ? AND created_at <= ?", teamID, from, to).
		Scan(&agg)

	s.db.WithContext(ctx).Model(&entities.RemediationAction{}).
		Where("team_id = ? AND created_at >= ? AND created_at <= ?", teamID, from, to).
		Count(&remediationsSuggested)
	s.db.WithContext(ctx).Model(&entities.RemediationAction{}).
		Where("team_id = ? AND status IN ? AND created_at >= ? AND created_at <= ?",
			teamID, []string{"EXECUTED", "VERIFIED"}, from, to).
		Count(&remediationsExecuted)

	s.db.WithContext(ctx).Model(&entities.AgentTriggerRule{}).
		Where("team_id = ? AND enabled = ?", teamID, true).
		Count(&activeTriggerRules)
	s.db.WithContext(ctx).Model(&entities.AgentTriggerRule{}).
		Where("team_id = ? AND product = ?", teamID, "log_search").
		Count(&logTriggerRules)

	// 5. Service catalog metrics for Log-Search & Pantomath
	var catalogItems []entities.ServiceCatalogItem
	s.db.WithContext(ctx).Where("team_id = ?", teamID).Find(&catalogItems)
	logServiceSet := make(map[string]struct{})
	pantoProductSet := make(map[string]struct{})
	var linkedRepos int64
	for _, item := range catalogItems {
		for _, ls := range item.LogSearchServices {
			if ls != "" {
				logServiceSet[ls] = struct{}{}
			}
		}
		if item.PantomathProduct != nil && *item.PantomathProduct != "" {
			pantoProductSet[*item.PantomathProduct] = struct{}{}
		}
		if item.RepositoryURL != nil && *item.RepositoryURL != "" {
			linkedRepos++
		}
	}

	// 6. Daily counter metrics (MCP calls)
	fromDateStr := from.UTC().Format("2006-01-02")
	toDateStr := to.UTC().Format("2006-01-02")
	var mcpCallsTotal, pantoMCPCalls int64
	s.db.WithContext(ctx).Model(&entities.UsageDailyCounter{}).
		Select("COALESCE(SUM(count), 0)").
		Where("team_id = ? AND metric_key = ? AND date >= ? AND date <= ?", teamID, "mcp_calls", fromDateStr, toDateStr).
		Scan(&mcpCallsTotal)
	s.db.WithContext(ctx).Model(&entities.UsageDailyCounter{}).
		Select("COALESCE(SUM(count), 0)").
		Where("team_id = ? AND product = ? AND date >= ? AND date <= ?", teamID, entities.ProductPantomath, fromDateStr, toDateStr).
		Scan(&pantoMCPCalls)

	// 7. Build zero-filled daily timeseries
	dates, byDate := buildDateSeries(from, to)

	// 7a. ClickHouse probe counts (total, daily, top monitors)
	var totalProbes int64
	topMonitors := make([]TopMonitorConsumer, 0)
	if s.ch != nil {
		type chDailyRow struct {
			Day   time.Time `gorm:"column:day"`
			Count int64     `gorm:"column:cnt"`
		}
		var chDaily []chDailyRow
		if err := s.ch.WithContext(ctx).Table("checks").
			Select("toStartOfDay(created_at) AS day, count() AS cnt").
			Where("team_id = ? AND created_at >= ? AND created_at <= ?", teamID, from, to).
			Group("day").
			Order("day ASC").
			Scan(&chDaily).Error; err != nil {
			s.logger.WithError(err).Warn("failed to query ClickHouse daily probe counts")
		} else {
			for _, r := range chDaily {
				dayKey := r.Day.UTC().Format("2006-01-02")
				totalProbes += r.Count
				if pt, ok := byDate[dayKey]; ok {
					pt.Probes = r.Count
				}
			}
		}

		type chMonitorRow struct {
			MonitorID uint64 `gorm:"column:monitor_id"`
			Count     int64  `gorm:"column:cnt"`
		}
		var chMonitors []chMonitorRow
		if err := s.ch.WithContext(ctx).Table("checks").
			Select("monitor_id, count() AS cnt").
			Where("team_id = ? AND created_at >= ? AND created_at <= ?", teamID, from, to).
			Group("monitor_id").
			Order("cnt DESC").
			Limit(10).
			Scan(&chMonitors).Error; err != nil {
			s.logger.WithError(err).Warn("failed to query ClickHouse top monitors")
		} else if len(chMonitors) > 0 {
			monitorIDs := make([]uint64, 0, len(chMonitors))
			for _, m := range chMonitors {
				monitorIDs = append(monitorIDs, m.MonitorID)
			}
			var monitors []entities.Monitor
			s.db.WithContext(ctx).Select("id, name").Where("team_id = ? AND id IN ?", teamID, monitorIDs).Find(&monitors)
			nameByID := make(map[uint64]string, len(monitors))
			for _, m := range monitors {
				nameByID[uint64(m.ID)] = m.Name
			}
			for _, m := range chMonitors {
				name := nameByID[m.MonitorID]
				if name == "" {
					name = "Deleted Monitor"
				}
				var share float64
				if totalProbes > 0 {
					share = math.Round((float64(m.Count)/float64(totalProbes))*1000) / 10
				}
				topMonitors = append(topMonitors, TopMonitorConsumer{
					MonitorID: m.MonitorID,
					Name:      name,
					Probes:    m.Count,
					SharePct:  share,
				})
			}
		}
	}

	// 7b. Postgres daily investigations & tokens
	type pgInvDayRow struct {
		Day    string `gorm:"column:day"`
		Runs   int64  `gorm:"column:runs"`
		Tokens int64  `gorm:"column:tokens"`
	}
	var invDays []pgInvDayRow
	s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
		Select("TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day, COUNT(*) AS runs, COALESCE(SUM(tokens_used), 0) AS tokens").
		Where("team_id = ? AND created_at >= ? AND created_at <= ?", teamID, from, to).
		Group("day").
		Scan(&invDays)
	for _, r := range invDays {
		if pt, ok := byDate[r.Day]; ok {
			pt.AgentRuns = r.Runs
			pt.AgentTokens = r.Tokens
		}
	}

	// 7c. Postgres daily MCP calls
	type pgCounterDayRow struct {
		Date  string `gorm:"column:date"`
		Count int64  `gorm:"column:cnt"`
	}
	var mcpDays []pgCounterDayRow
	s.db.WithContext(ctx).Model(&entities.UsageDailyCounter{}).
		Select("date, COALESCE(SUM(count), 0) AS cnt").
		Where("team_id = ? AND metric_key = ? AND date >= ? AND date <= ?", teamID, "mcp_calls", fromDateStr, toDateStr).
		Group("date").
		Scan(&mcpDays)
	for _, r := range mcpDays {
		if pt, ok := byDate[r.Date]; ok {
			pt.MCPCalls = r.Count
		}
	}

	// 7d. Postgres daily incidents & notifications
	type pgCountDayRow struct {
		Day   string `gorm:"column:day"`
		Count int64  `gorm:"column:cnt"`
	}
	var incDays []pgCountDayRow
	s.db.WithContext(ctx).Model(&entities.Incident{}).
		Select("TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day, COUNT(*) AS cnt").
		Where("team_id = ? AND created_at >= ? AND created_at <= ?", teamID, from, to).
		Group("day").
		Scan(&incDays)
	for _, r := range incDays {
		if pt, ok := byDate[r.Day]; ok {
			pt.Incidents = r.Count
		}
	}

	var notifDays []pgCountDayRow
	s.db.WithContext(ctx).Table("notification_jobs").
		Select("TO_CHAR(notification_jobs.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day, COUNT(*) AS cnt").
		Joins("JOIN incidents ON incidents.id = notification_jobs.incident_id").
		Where("incidents.team_id = ? AND notification_jobs.created_at >= ? AND notification_jobs.created_at <= ?", teamID, from, to).
		Group("day").
		Scan(&notifDays)
	for _, r := range notifDays {
		if pt, ok := byDate[r.Day]; ok {
			pt.Notifications = r.Count
		}
	}

	// 7e. Live Log-Search usage (events, bytes, top services, daily series)
	var logEventsIngested, logBytesIngested int64
	var liveLogIngestionGB float64
	var liveLogActiveServices int64
	topLogServices := make([]TopLogServiceConsumer, 0)
	if lsUsage := s.fetchLogSearchUsage(ctx, teamID, from, to); lsUsage != nil {
		logEventsIngested = lsUsage.EventsIngested
		logBytesIngested = lsUsage.BytesIngested
		liveLogIngestionGB = lsUsage.IngestionGB
		liveLogActiveServices = lsUsage.ActiveServices
		for _, svc := range lsUsage.TopServices {
			topLogServices = append(topLogServices, TopLogServiceConsumer{
				ServiceName: svc.ServiceName,
				Events:      svc.Events,
				Bytes:       svc.Bytes,
				SharePct:    svc.SharePct,
			})
		}
		for _, d := range lsUsage.Daily {
			if pt, ok := byDate[d.Date]; ok {
				pt.LogBytes = d.Bytes
				pt.LogEvents = d.Events
			}
		}
	}

	// 7f. Live Pantomath usage (indexed products, repos, vector chunks, graph edges) + downgrade lifecycle
	var livePantoProducts, livePantoRepos, pantoVectorChunks, pantoGraphEdges, pantoQueries int64
	topPantoProducts := make([]TopPantomathProductConsumer, 0)
	var discoveredPantoNames []string
	if pmUsage := s.fetchPantomathUsage(ctx, teamID); pmUsage != nil {
		livePantoProducts = pmUsage.IndexedProducts
		livePantoRepos = pmUsage.IndexedRepos
		pantoVectorChunks = pmUsage.VectorChunks
		pantoGraphEdges = pmUsage.GraphEdges
		pantoQueries = pmUsage.QueriesExecuted
		for _, p := range pmUsage.Products {
			discoveredPantoNames = append(discoveredPantoNames, p.Product)
			topPantoProducts = append(topPantoProducts, TopPantomathProductConsumer{
				Product:      p.Product,
				RepoCount:    p.RepoCount,
				VectorChunks: p.VectorChunks,
				Source:       p.Source,
				Indexing:     p.Indexing,
				Status:       "active",
			})
		}
	}
	if len(discoveredPantoNames) > 0 {
		gHelper := &Guard{db: s.db, usageSvc: s, httpClient: s.httpClient, logger: s.logger}
		_, lcMap := gHelper.ReconcilePantomathLifecycle(ctx, teamID, team.PaymentPlan, team.AllowOverage, discoveredPantoNames)
		for i := range topPantoProducts {
			if lc, ok := lcMap[topPantoProducts[i].Product]; ok {
				topPantoProducts[i].Status = strings.ToLower(string(lc.Status))
				topPantoProducts[i].DeleteAt = lc.DeleteAt
			}
		}
	}

	timeseries := make([]DailyUsagePoint, 0, len(dates))
	for _, d := range dates {
		timeseries = append(timeseries, *byDate[d])
	}

	// 8. Top agent triggers & services
	type triggerRow struct {
		TriggerSource string  `gorm:"column:trigger_source"`
		Runs          int64   `gorm:"column:runs"`
		Tokens        int64   `gorm:"column:tokens"`
		AvgDuration   float64 `gorm:"column:avg_dur"`
	}
	var triggerRows []triggerRow
	s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
		Select("trigger_source, COUNT(*) AS runs, COALESCE(SUM(tokens_used), 0) AS tokens, COALESCE(AVG(duration_ms), 0) AS avg_dur").
		Where("team_id = ? AND created_at >= ? AND created_at <= ?", teamID, from, to).
		Group("trigger_source").
		Order("runs DESC").
		Limit(10).
		Scan(&triggerRows)

	topTriggers := make([]TopTriggerConsumer, 0, len(triggerRows))
	for _, tr := range triggerRows {
		topTriggers = append(topTriggers, TopTriggerConsumer{
			TriggerSource: tr.TriggerSource,
			Runs:          tr.Runs,
			TokensUsed:    tr.Tokens,
			AvgDurationMs: int64(math.Round(tr.AvgDuration)),
		})
	}

	type serviceRow struct {
		ServiceName string `gorm:"column:service_name"`
		Runs        int64  `gorm:"column:runs"`
		Tokens      int64  `gorm:"column:tokens"`
	}
	var serviceRows []serviceRow
	s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
		Select("COALESCE(NULLIF(service_name, ''), 'Unassigned') AS service_name, COUNT(*) AS runs, COALESCE(SUM(tokens_used), 0) AS tokens").
		Where("team_id = ? AND created_at >= ? AND created_at <= ?", teamID, from, to).
		Group("service_name").
		Order("tokens DESC, runs DESC").
		Limit(10).
		Scan(&serviceRows)

	topServices := make([]TopServiceConsumer, 0, len(serviceRows))
	for _, sr := range serviceRows {
		topServices = append(topServices, TopServiceConsumer{
			ServiceName: sr.ServiceName,
			Runs:        sr.Runs,
			TokensUsed:  sr.Tokens,
		})
	}

	// Ensure deterministic ordering of topMonitors
	sort.SliceStable(topMonitors, func(i, j int) bool {
		return topMonitors[i].Probes > topMonitors[j].Probes
	})

	// 9. Assemble quotas
	monitorLimit := float64(team.GetMonitorLimit())
	statusPageLimit := float64(team.GetStatusPageLimit())
	escalationLimit := float64(escalationPolicyLimit(team.PaymentPlan))
	seatLimit := float64(team.GetTeamMemberLimit())

	agentUsed := float64(managedRuns)
	if float64(agentEnt.CurrentUsage) > agentUsed {
		agentUsed = float64(agentEnt.CurrentUsage)
	}
	agentLimit := float64(agentEnt.MonthlyQuota)
	if agentLimit == 0 {
		agentLimit = float64(team.GetManagedInvestigationLimit())
	}

	logUsedGB := float64(logEnt.CurrentUsage)
	if liveLogIngestionGB > logUsedGB {
		logUsedGB = liveLogIngestionGB
	}
	logLimitGB := float64(logEnt.MonthlyQuota)
	activeLogServices := int64(len(logServiceSet))
	if liveLogActiveServices > activeLogServices {
		activeLogServices = liveLogActiveServices
	}

	pantoUsed := float64(len(pantoProductSet))
	if float64(livePantoProducts) > pantoUsed {
		pantoUsed = float64(livePantoProducts)
	}
	if float64(pantoEnt.CurrentUsage) > pantoUsed {
		pantoUsed = float64(pantoEnt.CurrentUsage)
	}
	pantoLimit := float64(pantoEnt.MonthlyQuota)
	pantoRepos := linkedRepos
	if livePantoRepos > pantoRepos {
		pantoRepos = livePantoRepos
	}
	pantoTotalQueries := pantoMCPCalls
	if pantoQueries > pantoTotalQueries {
		pantoTotalQueries = pantoQueries
	}

	isPaid := team.PaymentPlan == entities.PaymentPlanPro || team.PaymentPlan == entities.PaymentPlanTeam || team.PaymentPlan == entities.PaymentPlanEnterprise
	cycleStart := cycle.Start.Format("2006-01-02")
	var storedStates []entities.QuotaNotificationState
	s.db.WithContext(ctx).
		Where("team_id = ? AND billing_cycle_start = ?", teamID, cycleStart).
		Find(&storedStates)
	graceByKey := make(map[string]*time.Time, len(storedStates))
	for i := range storedStates {
		graceByKey[string(storedStates[i].Product)+":"+storedStates[i].QuotaKey] = storedStates[i].GraceStartedAt
	}

	buildEnforcement := func(prod entities.ProductSlug, q QuotaItem, hint string) ProductEnforcementSummary {
		st, _, gEnd, overCents := ComputeEnforcementTransition(
			team.PaymentPlan, team.AllowOverage, team.OverageSpendCapCents,
			q.Key, q.Used, q.Limit, graceByKey[string(prod)+":"+q.Key], now,
		)
		var gHours *int
		if gEnd != nil && st == entities.QuotaStateGrace {
			h := int(math.Ceil(time.Until(*gEnd).Hours()))
			if h < 0 {
				h = 0
			}
			gHours = &h
		}
		return ProductEnforcementSummary{
			Product:               prod,
			QuotaKey:              q.Key,
			Label:                 q.Label,
			State:                 st,
			Used:                  q.Used,
			Limit:                 q.Limit,
			Unit:                  q.Unit,
			Percent:               q.Percent,
			GraceExpiresAt:        gEnd,
			GraceHoursRemaining:   gHours,
			EstimatedOverageCents: overCents,
			TopConsumerHint:       hint,
			BehaviorSummary:       DescribeEnforcementBehavior(prod, st),
		}
	}

	monQuota := buildQuotaItem("monitors", "Active Monitors", float64(monitorCount), monitorLimit, "monitors", nil)
	logQuota := buildQuotaItem("ingestion_gb", "Log Ingestion", logUsedGB, logLimitGB, "GB", projectMonthlyUsage(logUsedGB, cycle))
	agentQuota := buildQuotaItem("managed_investigations", "Managed AI Investigations", agentUsed, agentLimit, "runs", projectMonthlyUsage(agentUsed, cycle))
	agentTokenQuota := buildQuotaItem("monthly_tokens", "Monthly LLM Tokens", float64(agg.TotalTokens), float64(team.GetAgentMonthlyTokenLimit()), "tokens", projectMonthlyUsage(float64(agg.TotalTokens), cycle))
	agentTriggerQuota := buildQuotaItem("trigger_rules", "Active Trigger Rules", float64(activeTriggerRules), float64(team.GetAgentTriggerRuleLimit()), "rules", nil)
	pantoQuota := buildQuotaItem("indexed_products", "Indexed Products", pantoUsed, pantoLimit, "products", nil)
	pantoRepoQuota := buildQuotaItem("indexed_repos", "Indexed Repositories", float64(pantoRepos), float64(team.GetPantomathRepoLimit()), "repos", nil)
	pantoChunkQuota := buildQuotaItem("vector_chunks", "Graph Vector Chunks", float64(pantoVectorChunks), float64(team.GetPantomathVectorChunkLimit()), "chunks", nil)

	topLogHint := ""
	if len(topLogServices) > 0 {
		topLogHint = fmt.Sprintf("Top service: %s (%.1f%%)", topLogServices[0].ServiceName, topLogServices[0].SharePct)
	}
	enforcement := []ProductEnforcementSummary{
		buildEnforcement(entities.ProductMonitoring, monQuota, ""),
		buildEnforcement(entities.ProductLogSearch, logQuota, topLogHint),
		buildEnforcement(entities.ProductManagedAgent, agentQuota, ""),
		buildEnforcement(entities.ProductManagedAgent, agentTokenQuota, ""),
		buildEnforcement(entities.ProductPantomath, pantoQuota, ""),
		buildEnforcement(entities.ProductPantomath, pantoRepoQuota, ""),
		buildEnforcement(entities.ProductPantomath, pantoChunkQuota, ""),
	}
	totalOverageCents := 0
	for _, e := range enforcement {
		totalOverageCents += e.EstimatedOverageCents
	}

	resp := &TeamUsageResponse{
		TeamID:       teamID,
		Plan:         string(team.PaymentPlan),
		RangeStart:   from,
		RangeEnd:     to,
		BillingCycle: cycle,
		Overage: OverageSettings{
			AllowOverage:          team.AllowOverage,
			OverageSpendCapCents:  team.OverageSpendCapCents,
			EstimatedOverageCents: totalOverageCents,
			Eligible:              isPaid,
		},
		Enforcement: enforcement,
		Products: ProductsUsage{
			Monitoring: MonitoringUsage{
				Tier:    monEnt.Tier,
				Enabled: monEnt.Enabled,
				Quotas: []QuotaItem{
					monQuota,
					buildQuotaItem("status_pages", "Status Pages", float64(statusPageCount), statusPageLimit, "pages", nil),
					buildQuotaItem("escalation_policies", "Escalation Policies", float64(escalationPolicyCount), escalationLimit, "policies", nil),
				},
				ProbeExecutions:       totalProbes,
				ActiveHeartbeats:      heartbeatCount,
				NotificationsSent:     notificationsSent,
				IncidentsOpened:       incidentsOpened,
				StatusPageSubscribers: statusPageSubscribers,
			},
			LogSearch: LogSearchUsage{
				Tier:    logEnt.Tier,
				Enabled: logEnt.Enabled,
				Quotas: []QuotaItem{
					logQuota,
				},
				EventsIngested:             logEventsIngested,
				BytesIngested:              logBytesIngested,
				ActiveServices:             activeLogServices,
				TriggerRules:               logTriggerRules,
				LogTriggeredInvestigations: logTriggeredInv,
			},
			ManagedAgent: ManagedAgentUsage{
				Tier:    agentEnt.Tier,
				Enabled: agentEnt.Enabled,
				Quotas: []QuotaItem{
					agentQuota,
					agentTokenQuota,
					agentTriggerQuota,
				},
				ManagedRuns:           managedRuns,
				ByoaRuns:              byoaRuns,
				TokensUsed:            agg.TotalTokens,
				AvgDurationMs:         int64(math.Round(agg.AvgDuration)),
				MCPCalls:              mcpCallsTotal,
				RemediationsSuggested: remediationsSuggested,
				RemediationsExecuted:  remediationsExecuted,
				ActiveTriggerRules:    activeTriggerRules,
			},
			Pantomath: PantomathUsage{
				Tier:    pantoEnt.Tier,
				Enabled: pantoEnt.Enabled,
				Quotas: []QuotaItem{
					pantoQuota,
					pantoRepoQuota,
					pantoChunkQuota,
				},
				LinkedRepositories:               pantoRepos,
				VectorChunks:                     pantoVectorChunks,
				GraphEdges:                       pantoGraphEdges,
				QueriesExecuted:                  pantoTotalQueries,
				PantomathTriggeredInvestigations: pantoTriggeredInv,
				MCPCalls:                         pantoMCPCalls,
			},
		},
		Platform: PlatformUsage{
			TeamSeats:           buildQuotaItem("team_seats", "Team Seats", float64(activeMembers+pendingInvitations), seatLimit, "seats", nil),
			ActiveMembers:       activeMembers,
			PendingInvitations:  pendingInvitations,
			ServiceCatalogCount: serviceCatalogCount,
			ActiveAPIKeys:       activeAPIKeys,
			ReportsGenerated:    reportsGenerated,
		},
		Timeseries: timeseries,
		TopConsumers: TopConsumers{
			Monitors:          topMonitors,
			LogServices:       topLogServices,
			PantomathProducts: topPantoProducts,
			AgentTriggers:     topTriggers,
			Services:          topServices,
		},
	}

	return resp, nil
}

func (s *serviceImpl) UpdateOverageSettings(
	ctx context.Context,
	teamID uint,
	allowOverage bool,
	spendCapCents *int,
) (*OverageSettings, error) {
	var team entities.Team
	if err := s.db.WithContext(ctx).First(&team, teamID).Error; err != nil {
		return nil, err
	}
	isPaid := team.PaymentPlan == entities.PaymentPlanPro || team.PaymentPlan == entities.PaymentPlanTeam || team.PaymentPlan == entities.PaymentPlanEnterprise
	if allowOverage && !isPaid {
		return nil, fmt.Errorf("pay-as-you-go overages require a Team or Enterprise plan")
	}
	if spendCapCents != nil && *spendCapCents <= 0 {
		spendCapCents = nil
	}
	if err := s.db.WithContext(ctx).Model(&entities.Team{}).Where("id = ?", teamID).Updates(map[string]interface{}{
		"allow_overage":           allowOverage,
		"overage_spend_cap_cents": spendCapCents,
	}).Error; err != nil {
		return nil, err
	}
	return &OverageSettings{
		AllowOverage:         allowOverage,
		OverageSpendCapCents: spendCapCents,
		Eligible:             isPaid,
	}, nil
}
