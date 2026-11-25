package investigation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type AgentEventRequest struct {
	TeamID         uint                   `json:"teamId"`
	Product        entities.ProductSlug   `json:"product"`     // "monitoring", "log_search", "pantomath", "webhook"
	TriggerType    string                 `json:"triggerType"` // e.g. "logs.error_count_threshold", "logs.pattern_match", "pantomath.post_deploy_check"
	TriggerRuleID  *uint                  `json:"triggerRuleId,omitempty"`
	ServiceName    string                 `json:"serviceName"`
	Title          string                 `json:"title"`
	Summary        string                 `json:"summary"`
	CreateIncident bool                   `json:"createIncident,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Force          bool                   `json:"force,omitempty"`
	Sync           bool                   `json:"-"`
}

func (s *serviceImpl) ListTriggerRules(ctx context.Context, teamID uint) ([]entities.AgentTriggerRule, error) {
	var rules []entities.AgentTriggerRule
	if err := s.db.WithContext(ctx).Where("team_id = ?", teamID).Order("product asc, id asc").Find(&rules).Error; err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		defaults := defaultTriggerRulesForTeam(teamID)
		for i := range defaults {
			_ = s.db.WithContext(ctx).Create(&defaults[i]).Error
		}
		rules = defaults
	}
	return rules, nil
}

func defaultTriggerRulesForTeam(teamID uint) []entities.AgentTriggerRule {
	return []entities.AgentTriggerRule{
		{
			TeamID:          teamID,
			Product:         entities.ProductMonitoring,
			TriggerType:     "monitoring.target_down",
			Name:            "Target Down & Assertion Failures",
			Description:     "Automatically investigate confirmed uptime probe failures and HTTP assertion breaches.",
			Enabled:         true,
			Threshold:       3,
			WindowSeconds:   180,
			CooldownMinutes: 15,
			CreateIncident:  true,
		},
		{
			TeamID:          teamID,
			Product:         entities.ProductMonitoring,
			TriggerType:     "monitoring.latency_anomaly",
			Name:            "Latency Anomaly Streak",
			Description:     "Trigger agent triage when Forecaster detects 3+ consecutive anomalous probe latencies.",
			Enabled:         true,
			Threshold:       3,
			WindowSeconds:   300,
			CooldownMinutes: 15,
			CreateIncident:  true,
		},
		{
			TeamID:          teamID,
			Product:         entities.ProductLogSearch,
			TriggerType:     "logs.error_count_threshold",
			Name:            "Error Log Spike (>= 25 ERRORs / 5m)",
			Description:     "Trigger standalone log triage when a service emits 25 or more ERROR/FATAL logs within 5 minutes.",
			Enabled:         true,
			Threshold:       25,
			WindowSeconds:   300,
			CooldownMinutes: 15,
			CreateIncident:  false,
		},
		{
			TeamID:          teamID,
			Product:         entities.ProductLogSearch,
			TriggerType:     "logs.error_rate_spike",
			Name:            "High Error Rate (> 5% / 5m)",
			Description:     "Trigger investigation when a service's error rate exceeds 5% over a 5-minute window.",
			Enabled:         false,
			Threshold:       5,
			WindowSeconds:   300,
			CooldownMinutes: 15,
			CreateIncident:  false,
		},
		{
			TeamID:          teamID,
			Product:         entities.ProductLogSearch,
			TriggerType:     "logs.pattern_match",
			Name:            "Critical Pattern (OOM / Panic / Deadlock)",
			Description:     "Immediately investigate when critical crash signatures appear in application logs.",
			Enabled:         true,
			Threshold:       1,
			WindowSeconds:   300,
			Pattern:         "OOMKilled|panic:|SQLSTATE 53300|deadlock",
			CooldownMinutes: 15,
			CreateIncident:  true,
		},
		{
			TeamID:          teamID,
			Product:         entities.ProductPantomath,
			TriggerType:     "pantomath.post_deploy_check",
			Name:            "Post-Deploy Regression Check",
			Description:     "Correlate new errors and latency shifts after a repository push or deployment.",
			Enabled:         false,
			Threshold:       1,
			WindowSeconds:   300,
			CooldownMinutes: 30,
			CreateIncident:  false,
		},
	}
}

func (s *serviceImpl) ListActiveTriggerRulesByProduct(ctx context.Context, product entities.ProductSlug) ([]entities.AgentTriggerRule, error) {
	var rules []entities.AgentTriggerRule
	q := s.db.WithContext(ctx).Where("enabled = ?", true)
	if product != "" {
		q = q.Where("product = ?", product)
	}
	if err := q.Order("team_id asc, id asc").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *serviceImpl) CreateTriggerRule(ctx context.Context, rule *entities.AgentTriggerRule) (*entities.AgentTriggerRule, error) {
	if rule.WindowSeconds <= 0 {
		rule.WindowSeconds = 300
	}
	if rule.CooldownMinutes < 0 {
		rule.CooldownMinutes = 15
	}
	if err := s.db.WithContext(ctx).Create(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *serviceImpl) UpdateTriggerRule(ctx context.Context, teamID uint, ruleID uint, incoming *entities.AgentTriggerRule) (*entities.AgentTriggerRule, error) {
	var existing entities.AgentTriggerRule
	if err := s.db.WithContext(ctx).Where("id = ? AND team_id = ?", ruleID, teamID).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if incoming.Name != "" {
		existing.Name = incoming.Name
	}
	existing.Description = incoming.Description
	existing.Enabled = incoming.Enabled
	existing.ServiceFilter = incoming.ServiceFilter
	existing.Threshold = incoming.Threshold
	if incoming.WindowSeconds > 0 {
		existing.WindowSeconds = incoming.WindowSeconds
	}
	existing.Pattern = incoming.Pattern
	if incoming.CooldownMinutes >= 0 {
		existing.CooldownMinutes = incoming.CooldownMinutes
	}
	existing.CreateIncident = incoming.CreateIncident
	existing.RemediationLevelOverride = incoming.RemediationLevelOverride

	if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

func (s *serviceImpl) DeleteTriggerRule(ctx context.Context, teamID uint, ruleID uint) error {
	return s.db.WithContext(ctx).Where("id = ? AND team_id = ?", ruleID, teamID).Delete(&entities.AgentTriggerRule{}).Error
}

func (s *serviceImpl) TriggerFromEvent(ctx context.Context, req AgentEventRequest) (*entities.IncidentInvestigation, error) {
	if req.TeamID == 0 {
		return nil, errors.New("teamId is required")
	}

	agentCfg, err := s.GetAgentConfig(ctx, req.TeamID)
	if err != nil {
		return nil, err
	}
	if !agentCfg.Enabled && !req.Force {
		return nil, nil
	}

	// Load matched rule if provided
	var rule *entities.AgentTriggerRule
	if req.TriggerRuleID != nil && *req.TriggerRuleID > 0 {
		var r entities.AgentTriggerRule
		if err := s.db.WithContext(ctx).Where("id = ? AND team_id = ?", *req.TriggerRuleID, req.TeamID).First(&r).Error; err == nil {
			rule = &r
			if !r.Enabled && !req.Force {
				return nil, nil
			}
		}
	}

	// Gatekeeper for log_search events when no explicit rule override
	if req.Product == entities.ProductLogSearch && !agentCfg.AutoTriggerOnLogSpike && !req.Force {
		return nil, nil
	}

	serviceName := strings.TrimSpace(req.ServiceName)
	if serviceName == "" {
		serviceName = "unknown-service"
	}

	// Deduplicate in-flight investigations for same team + product + service
	if !req.Force {
		var inFlight int64
		s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
			Where("team_id = ? AND source_product = ? AND service_name = ? AND status IN ?",
				req.TeamID, string(req.Product), serviceName,
				[]entities.InvestigationStatus{entities.InvestigationStatusQueued, entities.InvestigationStatusRunning},
			).Count(&inFlight)
		if inFlight > 0 {
			return nil, nil
		}

		cooldownMins := agentCfg.CooldownMinutes
		if rule != nil && rule.CooldownMinutes > 0 {
			cooldownMins = rule.CooldownMinutes
		}
		if cooldownMins > 0 {
			cutoff := time.Now().Add(-time.Duration(cooldownMins) * time.Minute)
			var recent int64
			q := s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
				Where("team_id = ? AND source_product = ? AND service_name = ? AND created_at > ?",
					req.TeamID, string(req.Product), serviceName, cutoff)
			if req.TriggerType != "" {
				q = q.Where("trigger_type = ?", req.TriggerType)
			}
			q.Count(&recent)
			if recent > 0 {
				return nil, nil
			}
		}
	}

	// Check & increment managed_agent quota atomically (never bypassable via req.Force)
	effectiveMode, quotaErr := s.reserveManagedQuotaOrFallback(ctx, req.TeamID, agentCfg)
	if quotaErr != nil {
		return nil, quotaErr
	}

	// Optionally create a real Incident if rule or request asks for it
	var linkedIncident *entities.Incident
	var linkedIncidentID *uint
	shouldCreateIncident := req.CreateIncident || (rule != nil && rule.CreateIncident)
	if shouldCreateIncident && s.incidentService != nil {
		title := strings.TrimSpace(req.Title)
		if title == "" {
			title = fmt.Sprintf("[%s] %s on %s", strings.ToUpper(string(req.Product)), req.TriggerType, serviceName)
		}
		desc := strings.TrimSpace(req.Summary)
		incList := []entities.Incident{{
			TeamID:      req.TeamID,
			Title:       title,
			Description: &desc,
			Occurrences: 1,
		}}
		if err := s.incidentService.Create(ctx, &incList); err == nil && len(incList) > 0 && incList[0].ID != 0 {
			linkedIncident = &incList[0]
			idCopy := incList[0].ID
			linkedIncidentID = &idCopy
		}
	}

	ctxPayload, _ := json.Marshal(map[string]interface{}{
		"product":     req.Product,
		"triggerType": req.TriggerType,
		"title":       req.Title,
		"summary":     req.Summary,
		"serviceName": serviceName,
		"metadata":    req.Metadata,
	})

	triggerSource := "API"
	switch req.Product {
	case entities.ProductLogSearch:
		triggerSource = "LOG_SPIKE"
	case entities.ProductPantomath:
		triggerSource = "PANTOMATH_DEPLOY"
	case entities.ProductMonitoring:
		triggerSource = "AUTOMATIC"
	default:
		triggerSource = "WEBHOOK"
	}

	// Apply per-rule remediation override & effective execution mode
	effectiveCfg := *agentCfg
	effectiveCfg.ExecutionMode = effectiveMode
	if rule != nil && rule.RemediationLevelOverride != "" {
		effectiveCfg.RemediationLevel = rule.RemediationLevelOverride
	}

	startedAt := time.Now().UTC()
	initialStatus := entities.InvestigationStatusQueued
	if req.Sync {
		initialStatus = entities.InvestigationStatusRunning
	}

	var promptHint *string
	if strings.TrimSpace(req.Summary) != "" {
		s := strings.TrimSpace(req.Summary)
		if len(s) > 4000 {
			s = s[:4000]
		}
		promptHint = &s
	}

	inv := &entities.IncidentInvestigation{
		UUID:               "inv_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16],
		TeamID:             req.TeamID,
		IncidentID:         linkedIncidentID,
		SourceProduct:      string(req.Product),
		TriggerType:        req.TriggerType,
		TriggerRuleID:      req.TriggerRuleID,
		TriggerContextJSON: string(ctxPayload),
		ServiceName:        serviceName,
		TriggerSource:      triggerSource,
		ExecutionMode:      effectiveCfg.ExecutionMode,
		Status:             initialStatus,
		UserPrompt:         promptHint,
		StartedAt:          startedAt,
	}

	if err := s.db.WithContext(ctx).Create(inv).Error; err != nil {
		return nil, err
	}

	run := func(runCtx context.Context) {
		if effectiveCfg.ExecutionMode == entities.AgentModeByoaWebhook && effectiveCfg.ByoaWebhookURL != nil && *effectiveCfg.ByoaWebhookURL != "" {
			s.executeByoaWebhookFlow(runCtx, inv, linkedIncident, nil, &effectiveCfg)
			return
		}
		s.executeManagedAgentFlow(runCtx, inv, linkedIncident, nil, &effectiveCfg)
	}

	if req.Sync {
		run(ctx)
		var completed entities.IncidentInvestigation
		_ = s.db.WithContext(ctx).
			Preload("Steps", func(db *gorm.DB) *gorm.DB {
				return db.Order("step_index asc")
			}).
			Preload("RemediationActions").
			First(&completed, inv.ID).Error
		return &completed, nil
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), queueWaitTimeout+investigationTimeout)
		defer cancel()
		run(bgCtx)
	}()

	return inv, nil
}
