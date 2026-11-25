package usage

import (
	"context"
	"math"
	"sort"
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
	AgentRuns     int64  `json:"agentRuns"`
	AgentTokens   int64  `json:"agentTokens"`
	MCPCalls      int64  `json:"mcpCalls"`
	Notifications int64  `json:"notifications"`
	Incidents     int64  `json:"incidents"`
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

type TopConsumers struct {
	Monitors      []TopMonitorConsumer `json:"monitors"`
	AgentTriggers []TopTriggerConsumer `json:"agentTriggers"`
	Services      []TopServiceConsumer `json:"services"`
}

type TeamUsageResponse struct {
	TeamID       uint              `json:"teamId"`
	Plan         string            `json:"plan"`
	RangeStart   time.Time         `json:"rangeStart"`
	RangeEnd     time.Time         `json:"rangeEnd"`
	BillingCycle BillingCycleInfo  `json:"billingCycle"`
	Products     ProductsUsage     `json:"products"`
	Platform     PlatformUsage     `json:"platform"`
	Timeseries   []DailyUsagePoint `json:"timeseries"`
	TopConsumers TopConsumers      `json:"topConsumers"`
}

type Service interface {
	GetTeamUsage(ctx context.Context, teamID uint, from, to time.Time) (*TeamUsageResponse, error)
	IncrementCounter(ctx context.Context, teamID uint, product entities.ProductSlug, metricKey string, delta int64) error
}

type serviceImpl struct {
	db                   *gorm.DB
	ch                   *gorm.DB
	investigationService investigation.Service
	logger               *logrus.Entry
}

func NewService(
	db *gorm.DB,
	ch *gorm.DB,
	investigationService investigation.Service,
	logger *logrus.Entry,
) Service {
	return &serviceImpl{
		db:                   db,
		ch:                   ch,
		investigationService: investigationService,
		logger:               logger,
	}
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
	item := QuotaItem{
		Key:       key,
		Label:     label,
		Used:      math.Round(used*100) / 100,
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
	case entities.PaymentPlanTeam:
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

	entMap := make(map[entities.ProductSlug]entities.TeamEntitlement)
	if s.investigationService != nil {
		if ents, err := s.investigationService.GetEntitlements(ctx, teamID); err == nil {
			for _, e := range ents {
				entMap[e.Product] = e
			}
		}
	}

	getEnt := func(p entities.ProductSlug) entities.TeamEntitlement {
		if e, ok := entMap[p]; ok {
			return e
		}
		tier := entities.EntitlementTierFree
		switch team.PaymentPlan {
		case entities.PaymentPlanTeam:
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
	logLimitGB := float64(logEnt.MonthlyQuota)

	pantoUsed := float64(len(pantoProductSet))
	if float64(pantoEnt.CurrentUsage) > pantoUsed {
		pantoUsed = float64(pantoEnt.CurrentUsage)
	}
	pantoLimit := float64(pantoEnt.MonthlyQuota)

	resp := &TeamUsageResponse{
		TeamID:       teamID,
		Plan:         string(team.PaymentPlan),
		RangeStart:   from,
		RangeEnd:     to,
		BillingCycle: cycle,
		Products: ProductsUsage{
			Monitoring: MonitoringUsage{
				Tier:    monEnt.Tier,
				Enabled: monEnt.Enabled,
				Quotas: []QuotaItem{
					buildQuotaItem("monitors", "Active Monitors", float64(monitorCount), monitorLimit, "monitors", nil),
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
					buildQuotaItem("ingestion_gb", "Log Ingestion", logUsedGB, logLimitGB, "GB", projectMonthlyUsage(logUsedGB, cycle)),
				},
				ActiveServices:             int64(len(logServiceSet)),
				TriggerRules:               logTriggerRules,
				LogTriggeredInvestigations: logTriggeredInv,
			},
			ManagedAgent: ManagedAgentUsage{
				Tier:    agentEnt.Tier,
				Enabled: agentEnt.Enabled,
				Quotas: []QuotaItem{
					buildQuotaItem("managed_investigations", "Managed AI Investigations", agentUsed, agentLimit, "runs", projectMonthlyUsage(agentUsed, cycle)),
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
					buildQuotaItem("indexed_products", "Indexed Products", pantoUsed, pantoLimit, "products", nil),
				},
				LinkedRepositories:               linkedRepos,
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
			Monitors:      topMonitors,
			AgentTriggers: topTriggers,
			Services:      topServices,
		},
	}

	return resp, nil
}
