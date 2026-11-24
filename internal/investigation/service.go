package investigation

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/llm"
	"github.com/opsway-io/backend/internal/mcp"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	maxAgentIterations    = 8
	maxToolOutputForLLM   = 1600
	maxStepOutputSummary  = 1200
	investigationTimeout  = 3 * time.Minute
)

var (
	ErrQuotaExceeded = errors.New("monthly managed agent investigation quota exceeded; upgrade plan or switch to BYOA mode")
	ErrNotFound      = errors.New("record not found")
)

type TriggerRequest struct {
	TeamID        uint
	IncidentID    uint
	TriggerSource string // "AUTOMATIC", "MANUAL_UI", "MANUAL_SLACK", "LOG_SPIKE", "API"
	UserPrompt    *string
	Force         bool
	Sync          bool // When true, runs inline instead of background goroutine (useful for tests/CLI)
}

type ByoaCallbackPayload struct {
	Status             string                        `json:"status"`
	RootCauseSummary   string                        `json:"rootCauseSummary"`
	DetailedDiagnosis  string                        `json:"detailedDiagnosis"`
	ConfidenceScore    float64                       `json:"confidenceScore"`
	CorrelatedLogs     []entities.CorrelatedLogEntry `json:"correlatedLogs"`
	Steps              []entities.InvestigationStep  `json:"steps"`
	RemediationActions []entities.RemediationAction  `json:"remediationActions"`
}

type Service interface {
	// Entitlements
	GetEntitlements(ctx context.Context, teamID uint) ([]entities.TeamEntitlement, error)
	UpdateEntitlement(ctx context.Context, teamID uint, product entities.ProductSlug, tier entities.EntitlementTier, enabled bool) (*entities.TeamEntitlement, error)

	// Service Catalog
	ListServiceCatalog(ctx context.Context, teamID uint) ([]entities.ServiceCatalogItem, error)
	UpsertServiceCatalogItem(ctx context.Context, item *entities.ServiceCatalogItem) (*entities.ServiceCatalogItem, error)
	DeleteServiceCatalogItem(ctx context.Context, teamID uint, itemID uint) error

	// Agent Configuration
	GetAgentConfig(ctx context.Context, teamID uint) (*entities.AgentConfig, error)
	UpdateAgentConfig(ctx context.Context, teamID uint, cfg *entities.AgentConfig) (*entities.AgentConfig, error)

	// Investigations & Remediation
	ListInvestigationsByTeam(ctx context.Context, teamID uint, limit int) ([]entities.IncidentInvestigation, error)
	GetInvestigationsByIncident(ctx context.Context, teamID uint, incidentID uint) ([]entities.IncidentInvestigation, error)
	TriggerInvestigation(ctx context.Context, req TriggerRequest) (*entities.IncidentInvestigation, error)
	HandleByoaCallback(ctx context.Context, teamID uint, incidentID uint, invUUID string, payload ByoaCallbackPayload) (*entities.IncidentInvestigation, error)
	ApproveRemediationAction(ctx context.Context, teamID uint, investigationID uint, actionID uint, userID *uint) (*entities.IncidentInvestigation, error)
	RejectRemediationAction(ctx context.Context, teamID uint, investigationID uint, actionID uint) (*entities.IncidentInvestigation, error)
}

type serviceImpl struct {
	db                 *gorm.DB
	mcpGateway         mcp.Gateway
	incidentService    incident.Service
	monitorService     monitor.Service
	llmClient          llm.Client
	logSearchPublicURL string
	logger             *logrus.Entry
	httpClient         *http.Client
	managedGPUSem      chan struct{}
}

func NewService(
	db *gorm.DB,
	mcpGateway mcp.Gateway,
	incidentService incident.Service,
	monitorService monitor.Service,
	llmClient llm.Client,
	logSearchPublicURL string,
	logger *logrus.Entry,
) Service {
	if logSearchPublicURL == "" {
		logSearchPublicURL = "https://logs.opsway.eu"
	}
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}
	return &serviceImpl{
		db:                 db,
		mcpGateway:         mcpGateway,
		incidentService:    incidentService,
		monitorService:     monitorService,
		llmClient:          llmClient,
		logSearchPublicURL: strings.TrimRight(logSearchPublicURL, "/"),
		logger:             logger.WithField("component", "investigation_service"),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		// Serialize local managed GPU inference (4 GiB VRAM RTX 3050 Ti) so concurrent investigations don't OOM VRAM.
		managedGPUSem: make(chan struct{}, 1),
	}
}

func (s *serviceImpl) GetEntitlements(ctx context.Context, teamID uint) ([]entities.TeamEntitlement, error) {
	var entitlements []entities.TeamEntitlement
	if err := s.db.WithContext(ctx).Where("team_id = ?", teamID).Order("id asc").Find(&entitlements).Error; err != nil {
		return nil, err
	}

	if len(entitlements) == 0 {
		defaults := []entities.ProductSlug{
			entities.ProductMonitoring,
			entities.ProductLogSearch,
			entities.ProductPantomath,
			entities.ProductManagedAgent,
		}
		for _, prod := range defaults {
			ent := entities.TeamEntitlement{
				TeamID:       teamID,
				Product:      prod,
				Tier:         entities.EntitlementTierFree,
				Enabled:      true,
				MonthlyQuota: entities.DefaultMonthlyQuota(prod, entities.EntitlementTierFree),
				CurrentUsage: 0,
			}
			_ = s.db.WithContext(ctx).Create(&ent).Error
			entitlements = append(entitlements, ent)
		}
	}

	return entitlements, nil
}

func (s *serviceImpl) UpdateEntitlement(ctx context.Context, teamID uint, product entities.ProductSlug, tier entities.EntitlementTier, enabled bool) (*entities.TeamEntitlement, error) {
	var ent entities.TeamEntitlement
	err := s.db.WithContext(ctx).Where("team_id = ? AND product = ?", teamID, product).First(&ent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ent = entities.TeamEntitlement{
			TeamID:       teamID,
			Product:      product,
			Tier:         tier,
			Enabled:      enabled,
			MonthlyQuota: entities.DefaultMonthlyQuota(product, tier),
		}
		if err := s.db.WithContext(ctx).Create(&ent).Error; err != nil {
			return nil, err
		}
		return &ent, nil
	} else if err != nil {
		return nil, err
	}

	ent.Tier = tier
	ent.Enabled = enabled
	ent.MonthlyQuota = entities.DefaultMonthlyQuota(product, tier)
	if err := s.db.WithContext(ctx).Save(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

func (s *serviceImpl) ListServiceCatalog(ctx context.Context, teamID uint) ([]entities.ServiceCatalogItem, error) {
	var items []entities.ServiceCatalogItem
	if err := s.db.WithContext(ctx).Where("team_id = ?", teamID).Order("name asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *serviceImpl) UpsertServiceCatalogItem(ctx context.Context, item *entities.ServiceCatalogItem) (*entities.ServiceCatalogItem, error) {
	if item.ID != 0 {
		var existing entities.ServiceCatalogItem
		if err := s.db.WithContext(ctx).Where("id = ? AND team_id = ?", item.ID, item.TeamID).First(&existing).Error; err != nil {
			return nil, err
		}
		existing.Slug = item.Slug
		existing.Name = item.Name
		existing.Description = item.Description
		existing.LogSearchServices = item.LogSearchServices
		existing.MonitorIDs = item.MonitorIDs
		existing.RepositoryURL = item.RepositoryURL
		existing.PantomathProduct = item.PantomathProduct
		existing.EnvironmentLabels = item.EnvironmentLabels
		if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}

	if err := s.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *serviceImpl) DeleteServiceCatalogItem(ctx context.Context, teamID uint, itemID uint) error {
	return s.db.WithContext(ctx).Where("id = ? AND team_id = ?", itemID, teamID).Delete(&entities.ServiceCatalogItem{}).Error
}

func (s *serviceImpl) GetAgentConfig(ctx context.Context, teamID uint) (*entities.AgentConfig, error) {
	var cfg entities.AgentConfig
	err := s.db.WithContext(ctx).Where("team_id = ?", teamID).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = entities.AgentConfig{
			TeamID:                 teamID,
			Enabled:                true,
			ExecutionMode:          entities.AgentModeManaged,
			AutoTriggerOnConsensus: true,
			AutoTriggerOnAnomaly:   true,
			AutoTriggerOnLogSpike:  true,
			MinOccurrences:         1,
			CooldownMinutes:        30,
			RemediationLevel:       entities.RemediationApprovalRequired,
			ExternalMCPServers:     []entities.ExternalMCPServerConfig{},
			RemediationPlaybooks:   []entities.RemediationPlaybookConfig{},
		}
		_ = s.db.WithContext(ctx).Create(&cfg).Error
		return &cfg, nil
	} else if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *serviceImpl) UpdateAgentConfig(ctx context.Context, teamID uint, incoming *entities.AgentConfig) (*entities.AgentConfig, error) {
	cfg, err := s.GetAgentConfig(ctx, teamID)
	if err != nil {
		return nil, err
	}

	cfg.Enabled = incoming.Enabled
	cfg.ExecutionMode = incoming.ExecutionMode
	cfg.AutoTriggerOnConsensus = incoming.AutoTriggerOnConsensus
	cfg.AutoTriggerOnAnomaly = incoming.AutoTriggerOnAnomaly
	cfg.AutoTriggerOnLogSpike = incoming.AutoTriggerOnLogSpike
	if incoming.MinOccurrences > 0 {
		cfg.MinOccurrences = incoming.MinOccurrences
	}
	if incoming.CooldownMinutes >= 0 {
		cfg.CooldownMinutes = incoming.CooldownMinutes
	}
	if incoming.RemediationLevel != "" {
		cfg.RemediationLevel = incoming.RemediationLevel
	}
	cfg.ByoaWebhookURL = incoming.ByoaWebhookURL
	if incoming.ByoaWebhookSecret != nil && *incoming.ByoaWebhookSecret != "" && *incoming.ByoaWebhookSecret != "********" {
		cfg.ByoaWebhookSecret = incoming.ByoaWebhookSecret
	}
	cfg.CustomLLMProvider = incoming.CustomLLMProvider
	cfg.CustomLLMBaseURL = incoming.CustomLLMBaseURL
	cfg.CustomLLMModel = incoming.CustomLLMModel
	if incoming.CustomLLMAPIKey != nil && *incoming.CustomLLMAPIKey != "" && *incoming.CustomLLMAPIKey != "********" {
		cfg.CustomLLMAPIKey = incoming.CustomLLMAPIKey
	}
	if incoming.ExternalMCPServers != nil {
		cfg.ExternalMCPServers = incoming.ExternalMCPServers
	}
	if incoming.RemediationPlaybooks != nil {
		cfg.RemediationPlaybooks = incoming.RemediationPlaybooks
	}

	if err := s.db.WithContext(ctx).Save(cfg).Error; err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *serviceImpl) ListInvestigationsByTeam(ctx context.Context, teamID uint, limit int) ([]entities.IncidentInvestigation, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var list []entities.IncidentInvestigation
	err := s.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_index asc")
		}).
		Preload("RemediationActions").
		Where("team_id = ?", teamID).
		Order("created_at desc").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (s *serviceImpl) GetInvestigationsByIncident(ctx context.Context, teamID uint, incidentID uint) ([]entities.IncidentInvestigation, error) {
	var list []entities.IncidentInvestigation
	err := s.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_index asc")
		}).
		Preload("RemediationActions").
		Where("team_id = ? AND incident_id = ?", teamID, incidentID).
		Order("created_at desc").
		Find(&list).Error
	return list, err
}

func (s *serviceImpl) TriggerInvestigation(ctx context.Context, req TriggerRequest) (*entities.IncidentInvestigation, error) {
	inc, err := s.incidentService.GetByID(ctx, req.IncidentID)
	if err != nil || inc == nil || inc.TeamID != req.TeamID {
		return nil, ErrNotFound
	}

	agentCfg, err := s.GetAgentConfig(ctx, req.TeamID)
	if err != nil {
		return nil, err
	}

	// Gatekeeper check for automatic triggers
	if req.TriggerSource == "AUTOMATIC" && !req.Force {
		if !agentCfg.Enabled {
			return nil, nil
		}
		lowerTitle := strings.ToLower(inc.Title)
		if strings.Contains(lowerTitle, "cert expiry") || strings.Contains(lowerTitle, "deprecated tls") {
			return nil, nil
		}
		if strings.Contains(lowerTitle, "anomaly") && !agentCfg.AutoTriggerOnAnomaly {
			return nil, nil
		}
		if !strings.Contains(lowerTitle, "anomaly") && !agentCfg.AutoTriggerOnConsensus {
			return nil, nil
		}
		if inc.Occurrences < agentCfg.MinOccurrences {
			return nil, nil
		}

		if agentCfg.CooldownMinutes > 0 {
			cutoff := time.Now().Add(-time.Duration(agentCfg.CooldownMinutes) * time.Minute)
			var recentCount int64
			s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
				Where("team_id = ? AND incident_id = ? AND created_at > ?", req.TeamID, req.IncidentID, cutoff).
				Count(&recentCount)
			if recentCount > 0 {
				return nil, nil
			}
		}
	}

	// Resolve monitor & linked log-search service name
	serviceName := ""
	var monitorEntity *entities.Monitor
	if inc.MonitorID != nil && s.monitorService != nil {
		monitorEntity, _ = s.monitorService.GetMonitorAndSettingsByTeamIDAndID(ctx, req.TeamID, *inc.MonitorID)
		if monitorEntity != nil {
			if monitorEntity.Settings.InvestigationPolicy == "OFF" && req.TriggerSource == "AUTOMATIC" {
				return nil, nil
			}
			if monitorEntity.Settings.InvestigationPolicy == "MANUAL" && req.TriggerSource == "AUTOMATIC" {
				return nil, nil
			}
			if len(monitorEntity.Settings.LogSearchServices) > 0 && monitorEntity.Settings.LogSearchServices[0] != "" {
				serviceName = monitorEntity.Settings.LogSearchServices[0]
			} else if monitorEntity.Name != "" {
				serviceName = strings.ToLower(strings.ReplaceAll(monitorEntity.Name, " ", "-"))
			}
		}
	}

	// Check ServiceCatalogItem mapping if monitor is linked
	var catalogItems []entities.ServiceCatalogItem
	_ = s.db.WithContext(ctx).Where("team_id = ?", req.TeamID).Find(&catalogItems).Error
	for _, item := range catalogItems {
		if inc.MonitorID != nil {
			for _, mID := range item.MonitorIDs {
				if mID == *inc.MonitorID && len(item.LogSearchServices) > 0 {
					serviceName = item.LogSearchServices[0]
					break
				}
			}
		}
	}
	if serviceName == "" {
		serviceName = fmt.Sprintf("incident-%d", inc.ID)
	}

	// Check & increment quota if running in MANAGED mode
	if agentCfg.ExecutionMode == entities.AgentModeManaged {
		entitlements, _ := s.GetEntitlements(ctx, req.TeamID)
		for _, ent := range entitlements {
			if ent.Product == entities.ProductManagedAgent {
				if ent.MonthlyQuota != -1 && ent.CurrentUsage >= ent.MonthlyQuota && !req.Force {
					return nil, ErrQuotaExceeded
				}
				s.db.WithContext(ctx).Model(&entities.TeamEntitlement{}).
					Where("id = ?", ent.ID).
					UpdateColumn("current_usage", gorm.Expr("current_usage + 1"))
				break
			}
		}
	}

	startedAt := time.Now().UTC()
	initialStatus := entities.InvestigationStatusQueued
	if req.Sync {
		initialStatus = entities.InvestigationStatusRunning
	}

	inv := &entities.IncidentInvestigation{
		UUID:          "inv_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16],
		TeamID:        req.TeamID,
		IncidentID:    req.IncidentID,
		MonitorID:     inc.MonitorID,
		ServiceName:   serviceName,
		TriggerSource: req.TriggerSource,
		ExecutionMode: agentCfg.ExecutionMode,
		Status:        initialStatus,
		UserPrompt:    req.UserPrompt,
		StartedAt:     startedAt,
	}

	if err := s.db.WithContext(ctx).Create(inv).Error; err != nil {
		return nil, err
	}

	run := func(runCtx context.Context) {
		if agentCfg.ExecutionMode == entities.AgentModeByoaWebhook && agentCfg.ByoaWebhookURL != nil && *agentCfg.ByoaWebhookURL != "" {
			s.executeByoaWebhookFlow(runCtx, inv, inc, monitorEntity, agentCfg)
			return
		}
		s.executeManagedAgentFlow(runCtx, inv, inc, monitorEntity, agentCfg)
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
		bgCtx, cancel := context.WithTimeout(context.Background(), investigationTimeout)
		defer cancel()
		run(bgCtx)
	}()

	return inv, nil
}

func (s *serviceImpl) executeByoaWebhookFlow(
	ctx context.Context,
	inv *entities.IncidentInvestigation,
	inc *entities.Incident,
	mon *entities.Monitor,
	cfg *entities.AgentConfig,
) {
	inv.Status = entities.InvestigationStatusRunning
	_ = s.db.WithContext(ctx).Save(inv).Error

	stepStart := time.Now()
	webhookPayload := map[string]interface{}{
		"event":            "incident.investigation.requested",
		"investigation_id": inv.UUID,
		"team_id":          inv.TeamID,
		"tenant_id":        fmt.Sprintf("team_%d", inv.TeamID),
		"incident": map[string]interface{}{
			"id":          inc.ID,
			"title":       inc.Title,
			"description": inc.Description,
			"occurrences": inc.Occurrences,
			"created_at":  inc.CreatedAt,
		},
		"service_name": inv.ServiceName,
		"user_prompt":  inv.UserPrompt,
		"mcp_servers": map[string]interface{}{
			"opsway_unified_gateway": "https://api.opsway.eu/v1/mcp",
			"log_search_direct":      fmt.Sprintf("%s/mcp", s.logSearchPublicURL),
		},
		"callback_url": fmt.Sprintf("https://api.opsway.eu/v1/teams/%d/incidents/%d/investigations/%s/callback", inv.TeamID, inc.ID, inv.UUID),
	}

	payloadBytes, _ := json.Marshal(webhookPayload)
	dispatchSummary := fmt.Sprintf("Dispatched signed BYOA investigation request to %s.", *cfg.ByoaWebhookURL)
	stepStatus := "COMPLETED"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *cfg.ByoaWebhookURL, bytes.NewReader(payloadBytes))
	if err != nil {
		stepStatus = "FAILED"
		dispatchSummary = fmt.Sprintf("Failed to construct BYOA webhook request: %v", err)
	} else {
		req.Header.Set("Content-Type", "application/json")
		if cfg.ByoaWebhookSecret != nil && *cfg.ByoaWebhookSecret != "" {
			mac := hmac.New(sha256.New, []byte(*cfg.ByoaWebhookSecret))
			mac.Write(payloadBytes)
			req.Header.Set("X-Opsway-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		}
		resp, doErr := s.httpClient.Do(req)
		if doErr != nil {
			stepStatus = "FAILED"
			dispatchSummary = fmt.Sprintf("BYOA webhook dispatch to %s failed: %v", *cfg.ByoaWebhookURL, doErr)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				dispatchSummary += fmt.Sprintf(" Remote agent accepted request (HTTP %d). Awaiting callback.", resp.StatusCode)
			} else {
				stepStatus = "FAILED"
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
				dispatchSummary += fmt.Sprintf(" Remote agent returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			}
		}
	}

	_ = s.db.WithContext(ctx).Create(&entities.InvestigationStep{
		InvestigationID: inv.ID,
		StepIndex:       1,
		StepType:        "MCP_TOOL_CALL",
		ServerName:      "byoa-webhook",
		ToolName:        "byoa.dispatch_incident_webhook",
		InputJSON:       string(payloadBytes),
		OutputSummary:   dispatchSummary,
		Status:          stepStatus,
		DurationMs:      time.Since(stepStart).Milliseconds(),
	}).Error

	if stepStatus == "FAILED" {
		now := time.Now().UTC()
		inv.Status = entities.InvestigationStatusFailed
		inv.RootCauseSummary = &dispatchSummary
		inv.CompletedAt = &now
		inv.DurationMs = now.Sub(inv.StartedAt).Milliseconds()
		_ = s.db.WithContext(ctx).Save(inv).Error
	}
}

type agentRemediationProposal struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	ActionType    string `json:"actionType"`
	TargetService string `json:"targetService"`
	DiffOrCommand string `json:"diffOrCommand"`
	PayloadJSON   string `json:"payloadJson"`
}

type agentSynthesisResult struct {
	RootCauseSummary   string                     `json:"rootCauseSummary"`
	DetailedDiagnosis  string                     `json:"detailedDiagnosis"`
	ConfidenceScore    float64                    `json:"confidenceScore"`
	RemediationActions []agentRemediationProposal `json:"remediationActions"`
}

func (s *serviceImpl) executeManagedAgentFlow(
	ctx context.Context,
	inv *entities.IncidentInvestigation,
	inc *entities.Incident,
	mon *entities.Monitor,
	cfg *entities.AgentConfig,
) {
	// Serialize local managed GPU investigations so 4 GiB VRAM RTX 3050 Ti never OOMs.
	if cfg.ExecutionMode == entities.AgentModeManaged {
		select {
		case s.managedGPUSem <- struct{}{}:
			defer func() { <-s.managedGPUSem }()
		case <-ctx.Done():
			s.markInvestigationFailed(inv, fmt.Sprintf("Investigation timed out waiting for GPU slot: %v", ctx.Err()))
			return
		}
	}

	inv.Status = entities.InvestigationStatusRunning
	_ = s.db.WithContext(ctx).Save(inv).Error

	activeLLM := llm.NewClientForConfig(cfg, s.llmClient)
	if activeLLM == nil {
		s.markInvestigationFailed(inv, "No LLM client configured for managed or BYO_LLM investigation.")
		return
	}

	// Build focused tool catalog for compact 8k-16k context windows (4 GiB VRAM friendly).
	toolDefs, externalTools := s.buildAgentTools(ctx, inv.TeamID, cfg)

	incidentContext := s.buildIncidentPrompt(inv, inc, mon)
	messages := []llm.Message{
		{
			Role:    "system",
			Content: buildAgentSystemPrompt(),
		},
		{
			Role:    "user",
			Content: incidentContext,
		},
	}

	stepIdx := 1
	totalTokens := 0
	var correlatedLogs []entities.CorrelatedLogEntry
	var finalContent string
	var llmErr error

	hasLogSearchTool := false
	hasPantomathTool := false
	for _, td := range toolDefs {
		if strings.HasPrefix(td.Function.Name, "logs.") {
			hasLogSearchTool = true
		}
		if strings.HasPrefix(td.Function.Name, "pantomath.") {
			hasPantomathTool = true
		}
	}
	calledLogSearch := false
	nudgedLogSearch := false
	calledPantomath := false
	nudgedPantomath := false

	temp := 0.1
	for iter := 0; iter < maxAgentIterations; iter++ {
		callStart := time.Now()
		resp, err := activeLLM.Chat(ctx, llm.ChatCompletionRequest{
			Messages:    messages,
			Tools:       toolDefs,
			ToolChoice:  "auto",
			Temperature: &temp,
			MaxTokens:   1024,
		})
		if err != nil {
			llmErr = err
			s.logger.WithError(err).WithField("investigation_uuid", inv.UUID).Warn("LLM chat turn failed")
			break
		}

		totalTokens += resp.Usage.TotalTokens
		assistantMsg := resp.Choices[0].Message

		if len(assistantMsg.ToolCalls) == 0 {
			if hasPantomathTool && !calledPantomath && !nudgedPantomath && iter < maxAgentIterations-2 {
				nudgedPantomath = true
				targetHint := inv.ServiceName
				if mon != nil && len(mon.Steps) > 0 && strings.TrimSpace(mon.Steps[0].URL) != "" {
					targetHint = fmt.Sprintf("%s %s", inv.ServiceName, mon.Steps[0].URL)
				}
				messages = append(messages, assistantMsg, llm.Message{
					Role:    "user",
					Content: fmt.Sprintf("Before finalizing, call pantomath.ask_codebase with a descriptive multi-word query (for example: \"%s kubernetes deployment pods and backend dependencies\") so you can identify all candidate services and workloads to inspect for logs.", targetHint),
				})
				continue
			}
			if hasLogSearchTool && !calledLogSearch && !nudgedLogSearch && iter < maxAgentIterations-1 {
				nudgedLogSearch = true
				messages = append(messages, assistantMsg, llm.Message{
					Role:    "user",
					Content: fmt.Sprintf("Before finalizing your diagnosis, you MUST check application logs using logs.get_error_summary (and logs.semantic_search if errors exist) for service `%s` and any relevant backing services discovered from Pantomath.", inv.ServiceName),
				})
				continue
			}
			finalContent = strings.TrimSpace(assistantMsg.Content)
			break
		}

		messages = append(messages, assistantMsg)

		for _, tc := range assistantMsg.ToolCalls {
			if strings.HasPrefix(tc.Function.Name, "logs.") {
				calledLogSearch = true
			}
			if strings.HasPrefix(tc.Function.Name, "pantomath.") {
				calledPantomath = true
			}
			toolStart := time.Now()
			argsMap := parseToolArguments(tc.Function.Arguments)
			toolRes, callErr := s.invokeAgentTool(ctx, inv.TeamID, tc.Function.Name, argsMap, externalTools)

			toolText := extractFirstText(toolRes)
			stepStatus := "COMPLETED"
			if callErr != nil {
				stepStatus = "FAILED"
				toolText = fmt.Sprintf("Tool error: %v", callErr)
			} else if toolRes != nil && toolRes.IsError {
				stepStatus = "FAILED"
			}
			if strings.TrimSpace(toolText) == "" {
				toolText = "(empty tool response)"
			}

			// Extract real correlated log entries from log-search MCP tool outputs.
			if strings.HasPrefix(tc.Function.Name, "logs.") && stepStatus == "COMPLETED" {
				extracted := s.extractCorrelatedLogs(toolText, inv.ServiceName)
				correlatedLogs = appendUniqueLogs(correlatedLogs, extracted, 10)
			}

			_ = s.db.WithContext(ctx).Create(&entities.InvestigationStep{
				InvestigationID: inv.ID,
				StepIndex:       stepIdx,
				StepType:        "MCP_TOOL_CALL",
				ServerName:      inferServerName(tc.Function.Name),
				ToolName:        tc.Function.Name,
				InputJSON:       truncateText(tc.Function.Arguments, 1000),
				OutputSummary:   truncateText(toolText, maxStepOutputSummary),
				Status:          stepStatus,
				DurationMs:      time.Since(toolStart).Milliseconds(),
			}).Error
			stepIdx++

			messages = append(messages, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    truncateText(toolText, maxToolOutputForLLM),
			})
		}

		_ = callStart
	}

	// If loop exhausted max iterations without final synthesis, request one final JSON synthesis without tools.
	if finalContent == "" && llmErr == nil {
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: "You have reached the tool call budget. Based strictly on the evidence collected above, output the final JSON diagnosis object now.",
		})
		resp, err := activeLLM.Chat(ctx, llm.ChatCompletionRequest{
			Messages:    messages,
			Temperature: &temp,
			MaxTokens:   1024,
		})
		if err != nil {
			llmErr = err
		} else {
			totalTokens += resp.Usage.TotalTokens
			finalContent = strings.TrimSpace(resp.Choices[0].Message.Content)
		}
	}

	if llmErr != nil {
		errMsg := fmt.Sprintf("Agent LLM inference failed (%s): %v", activeLLM.Model(), llmErr)
		_ = s.db.WithContext(ctx).Create(&entities.InvestigationStep{
			InvestigationID: inv.ID,
			StepIndex:       stepIdx,
			StepType:        "SYNTHESIS",
			ServerName:      "opsway-agent",
			ToolName:        "agent.synthesize_diagnosis",
			OutputSummary:   errMsg,
			Status:          "FAILED",
			DurationMs:      0,
		}).Error
		inv.TokensUsed = totalTokens
		inv.CorrelatedLogs = correlatedLogs
		s.markInvestigationFailed(inv, errMsg)
		return
	}

	synthesis := parseSynthesisResult(finalContent, inv.ServiceName)
	if synthesis.ConfidenceScore <= 0 || synthesis.ConfidenceScore > 1 {
		synthesis.ConfidenceScore = 0.70
	}

	inv.RootCauseSummary = &synthesis.RootCauseSummary
	inv.DetailedDiagnosis = &synthesis.DetailedDiagnosis
	inv.ConfidenceScore = &synthesis.ConfidenceScore
	inv.CorrelatedLogs = correlatedLogs
	inv.TokensUsed = totalTokens

	_ = s.db.WithContext(ctx).Create(&entities.InvestigationStep{
		InvestigationID: inv.ID,
		StepIndex:       stepIdx,
		StepType:        "SYNTHESIS",
		ServerName:      "opsway-agent",
		ToolName:        "agent.synthesize_diagnosis",
		OutputSummary:   truncateText(synthesis.RootCauseSummary, maxStepOutputSummary),
		Status:          "COMPLETED",
		DurationMs:      0,
	}).Error
	stepIdx++

	// Build remediation actions from LLM proposals + matching configured playbooks.
	actionStatus := "PENDING_APPROVAL"
	if cfg.RemediationLevel == entities.RemediationAdvisory {
		actionStatus = "SUGGESTED"
	}

	createdActions := make([]entities.RemediationAction, 0, len(synthesis.RemediationActions)+2)
	for _, prop := range synthesis.RemediationActions {
		if strings.TrimSpace(prop.Title) == "" {
			continue
		}
		targetSvc := strings.TrimSpace(prop.TargetService)
		if targetSvc == "" {
			targetSvc = inv.ServiceName
		}
		actType := strings.TrimSpace(prop.ActionType)
		if actType == "" {
			actType = "CLI_COMMAND"
		}
		act := entities.RemediationAction{
			InvestigationID: inv.ID,
			TeamID:          inv.TeamID,
			IncidentID:      inc.ID,
			Title:           prop.Title,
			Description:     prop.Description,
			ActionType:      actType,
			TargetService:   targetSvc,
			DiffOrCommand:   prop.DiffOrCommand,
			PayloadJSON:     prop.PayloadJSON,
			Status:          actionStatus,
		}
		if err := s.db.WithContext(ctx).Create(&act).Error; err == nil {
			createdActions = append(createdActions, act)
		}
	}

	// Match team's configured RemediationPlaybooks against incident title + root cause.
	matchCorpus := inc.Title + "\n" + synthesis.RootCauseSummary
	for _, pb := range cfg.RemediationPlaybooks {
		if pb.TriggerPattern == "" {
			continue
		}
		re, err := regexp.Compile("(?i)" + pb.TriggerPattern)
		if err != nil || !re.MatchString(matchCorpus) {
			continue
		}
		pbPayload, _ := json.Marshal(map[string]interface{}{
			"playbook_id": pb.ID,
			"target_url":  pb.TargetURL,
			"method":      pb.Method,
			"service":     inv.ServiceName,
			"incident_id": inc.ID,
		})
		act := entities.RemediationAction{
			InvestigationID: inv.ID,
			TeamID:          inv.TeamID,
			IncidentID:      inc.ID,
			Title:           pb.Name,
			Description:     fmt.Sprintf("Matched allowlisted playbook `%s` (pattern `%s`).", pb.ID, pb.TriggerPattern),
			ActionType:      pb.ActionType,
			TargetService:   inv.ServiceName,
			DiffOrCommand:   fmt.Sprintf("%s %s", defaultHTTPMethod(pb.Method), pb.TargetURL),
			PayloadJSON:     string(pbPayload),
			Status:          actionStatus,
		}
		if err := s.db.WithContext(ctx).Create(&act).Error; err == nil {
			createdActions = append(createdActions, act)
		}
	}

	if cfg.RemediationLevel == entities.RemediationAutonomous && len(createdActions) > 0 {
		for i := range createdActions {
			if isExecutableActionType(createdActions[i].ActionType) {
				s.executeAndVerifyAction(ctx, inv, &createdActions[i], cfg, &stepIdx)
			}
		}
		inv.Status = entities.InvestigationStatusCompleted
	} else if cfg.RemediationLevel == entities.RemediationApprovalRequired && len(createdActions) > 0 {
		inv.Status = entities.InvestigationStatusAwaitingApproval
	} else {
		inv.Status = entities.InvestigationStatusCompleted
	}

	completedAt := time.Now().UTC()
	inv.CompletedAt = &completedAt
	inv.DurationMs = completedAt.Sub(inv.StartedAt).Milliseconds()
	_ = s.db.WithContext(ctx).Save(inv).Error

	// Sync RootCauseSummary to Incident.RootCauseAnalysis for backward compatibility.
	inc.RootCauseAnalysis = &synthesis.RootCauseSummary
	_ = s.incidentService.Update(ctx, inc)
}

func (s *serviceImpl) markInvestigationFailed(inv *entities.IncidentInvestigation, reason string) {
	now := time.Now().UTC()
	inv.Status = entities.InvestigationStatusFailed
	inv.RootCauseSummary = &reason
	inv.CompletedAt = &now
	inv.DurationMs = now.Sub(inv.StartedAt).Milliseconds()
	_ = s.db.WithContext(context.Background()).Save(inv).Error
}

func buildAgentSystemPrompt() string {
	return `You are Opsway's Autonomous SRE Agent. Investigate the production incident using the available MCP tools and return a grounded, factual Root Cause Analysis.

Rules:
1. Never invent log lines, SQL errors, Kubernetes deployments, or metrics that were not returned by a tool.
2. Follow this investigation workflow before finalizing:
   a) Inspect monitoring incident/probe timings (monitoring.get_incident_details / monitoring.get_monitor_check_timings).
   b) Query pantomath.ask_codebase with a descriptive multi-word query (including the service name, target URL/path, and "kubernetes deployment dependencies" — never a single word like "api") to map the failing monitor to its owning Kubernetes namespaces, deployments, pods, and upstream/downstream dependencies so you know where to look for logs.
   c) Correlate application logs via log-search (logs.list_services if exact service labels are unclear, logs.get_error_summary, and logs.semantic_search / logs.get_log_context if errors are found) across both the primary service and any suspect backing services discovered by Pantomath.
3. When you have enough evidence, stop calling tools and respond ONLY with a valid JSON object matching this exact schema:
{
  "rootCauseSummary": "Concise 2-3 sentence root cause grounded in tool outputs.",
  "detailedDiagnosis": "Markdown breakdown citing exact tool findings (probe latencies/status codes, log signatures, or codebase references).",
  "confidenceScore": 0.85,
  "remediationActions": [
    {
      "title": "Short action title",
      "description": "What this fixes and why",
      "actionType": "CLI_COMMAND | K8S_ROLLOUT_RESTART | WEBHOOK_CALL | GITHUB_PR",
      "targetService": "service-name",
      "diffOrCommand": "Exact command or patch snippet",
      "payloadJson": "{}"
    }
  ]
}`
}

func (s *serviceImpl) buildIncidentPrompt(inv *entities.IncidentInvestigation, inc *entities.Incident, mon *entities.Monitor) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Incident ID: %d\n", inc.ID))
	sb.WriteString(fmt.Sprintf("Title: %s\n", inc.Title))
	if inc.Description != nil && *inc.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", *inc.Description))
	}
	sb.WriteString(fmt.Sprintf("Occurrences: %d\n", inc.Occurrences))
	sb.WriteString(fmt.Sprintf("Created At: %s\n", inc.CreatedAt.UTC().Format(time.RFC3339)))
	if inc.MonitorID != nil {
		sb.WriteString(fmt.Sprintf("Monitor ID: %d\n", *inc.MonitorID))
	}
	if mon != nil && mon.Name != "" {
		sb.WriteString(fmt.Sprintf("Monitor Name: %s\n", mon.Name))
	}
	if mon != nil && len(mon.Steps) > 0 && strings.TrimSpace(mon.Steps[0].URL) != "" {
		sb.WriteString(fmt.Sprintf("Monitor Target URL: %s %s\n", mon.Steps[0].Method, mon.Steps[0].URL))
	}
	sb.WriteString(fmt.Sprintf("Service Name: %s\n", inv.ServiceName))
	if inv.UserPrompt != nil && strings.TrimSpace(*inv.UserPrompt) != "" {
		sb.WriteString(fmt.Sprintf("Operator Hint: %s\n", strings.TrimSpace(*inv.UserPrompt)))
	}
	sb.WriteString("\nInvestigate this incident using the MCP tools and produce the final JSON diagnosis.")
	return sb.String()
}

var coreAgentToolAllowlist = map[string]bool{
	"monitoring.get_incident_details":      true,
	"monitoring.get_monitor_check_timings": true,
	"monitoring.list_active_incidents":     true,
	"logs.list_services":                   true,
	"logs.get_error_summary":               true,
	"logs.semantic_search":                 true,
	"logs.get_log_context":                 true,
	"logs.tail_recent_logs":                true,
	"pantomath.ask_codebase":               true,
}

type externalToolRoute struct {
	server   entities.ExternalMCPServerConfig
	origName string
}

func (s *serviceImpl) buildAgentTools(
	ctx context.Context,
	teamID uint,
	cfg *entities.AgentConfig,
) ([]llm.ToolDefinition, map[string]externalToolRoute) {
	defs := make([]llm.ToolDefinition, 0, 10)
	if s.mcpGateway != nil {
		for _, t := range s.mcpGateway.ListTools(ctx, teamID) {
			if !coreAgentToolAllowlist[t.Name] {
				continue
			}
			defs = append(defs, llm.ToolDefinition{
				Type: "function",
				Function: llm.FunctionDefinition{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.InputSchema,
				},
			})
		}
	}

	extRoutes := make(map[string]externalToolRoute)
	for _, srv := range cfg.ExternalMCPServers {
		if !srv.Enabled || strings.TrimSpace(srv.URL) == "" {
			continue
		}
		tools := s.discoverExternalMCPTools(ctx, srv)
		for _, t := range tools {
			qualified := fmt.Sprintf("ext.%s.%s", sanitizeIdentifier(srv.Name), t.Name)
			extRoutes[qualified] = externalToolRoute{server: srv, origName: t.Name}
			defs = append(defs, llm.ToolDefinition{
				Type: "function",
				Function: llm.FunctionDefinition{
					Name:        qualified,
					Description: t.Description,
					Parameters:  t.InputSchema,
				},
			})
		}
	}
	return defs, extRoutes
}

func (s *serviceImpl) discoverExternalMCPTools(ctx context.Context, srv entities.ExternalMCPServerConfig) []mcp.Tool {
	rpcReq := mcp.JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "tools/list"}
	bodyBytes, _ := json.Marshal(rpcReq)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	if srv.AuthHeader != "" {
		req.Header.Set("Authorization", srv.AuthHeader)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var rpcResp struct {
		Result struct {
			Tools []mcp.Tool `json:"tools"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil
	}
	if len(rpcResp.Result.Tools) > 6 {
		return rpcResp.Result.Tools[:6]
	}
	return rpcResp.Result.Tools
}

func (s *serviceImpl) invokeAgentTool(
	ctx context.Context,
	teamID uint,
	toolName string,
	args map[string]interface{},
	extRoutes map[string]externalToolRoute,
) (*mcp.ToolCallResult, error) {
	if route, ok := extRoutes[toolName]; ok {
		return s.callExternalMCPTool(ctx, route.server, route.origName, args)
	}
	if s.mcpGateway == nil {
		return nil, errors.New("mcp gateway not configured")
	}
	return s.mcpGateway.CallTool(ctx, teamID, toolName, args)
}

func (s *serviceImpl) callExternalMCPTool(
	ctx context.Context,
	srv entities.ExternalMCPServerConfig,
	toolName string,
	args map[string]interface{},
) (*mcp.ToolCallResult, error) {
	paramsBytes, _ := json.Marshal(mcp.ToolCallParams{Name: toolName, Arguments: args})
	rpcReq := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  paramsBytes,
	}
	bodyBytes, _ := json.Marshal(rpcReq)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if srv.AuthHeader != "" {
		req.Header.Set("Authorization", srv.AuthHeader)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("external MCP %s returned HTTP %d: %s", srv.Name, resp.StatusCode, string(respBytes))
	}
	var rpcResp struct {
		Result *mcp.ToolCallResult `json:"result"`
		Error  *mcp.JSONRPCError   `json:"error"`
	}
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return &mcp.ToolCallResult{Content: []mcp.ContentItem{{Type: "text", Text: string(respBytes)}}}, nil
	}
	if rpcResp.Error != nil {
		return nil, errors.New(rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

func (s *serviceImpl) extractCorrelatedLogs(toolText string, serviceName string) []entities.CorrelatedLogEntry {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(toolText), &parsed); err != nil {
		return nil
	}

	var rawLogs []interface{}
	for _, key := range []string{"logs", "results", "matches", "entries"} {
		if arr, ok := parsed[key].([]interface{}); ok && len(arr) > 0 {
			rawLogs = arr
			break
		}
	}
	if len(rawLogs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	deepLink := fmt.Sprintf("%s/?query=%s&timeRange=1h&start_ts=%d&end_ts=%d",
		s.logSearchPublicURL,
		url.QueryEscape(fmt.Sprintf(`{service="%s"}`, serviceName)),
		now.Add(-30*time.Minute).UnixMilli(),
		now.UnixMilli(),
	)

	out := make([]entities.CorrelatedLogEntry, 0, len(rawLogs))
	for _, item := range rawLogs {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		msg := firstNonEmptyString(m, "message", "body", "log", "text")
		if msg == "" {
			continue
		}
		svc := firstNonEmptyString(m, "service_name", "serviceName", "service")
		if svc == "" {
			svc = serviceName
		}
		level := firstNonEmptyString(m, "log_level", "logLevel", "level", "severity")
		if level == "" {
			level = "INFO"
		}
		ts := firstNonEmptyString(m, "timestamp", "time", "created_at")
		if ts == "" {
			ts = now.Format(time.RFC3339)
		}
		logID := firstNonEmptyString(m, "id", "log_id", "logId")
		if logID == "" {
			logID = fmt.Sprintf("log_%d", len(out)+1)
		}
		labels := map[string]string{}
		if rawLabels, ok := m["labels"].(map[string]interface{}); ok {
			for k, v := range rawLabels {
				labels[k] = fmt.Sprintf("%v", v)
			}
		}
		out = append(out, entities.CorrelatedLogEntry{
			LogID:       logID,
			Timestamp:   ts,
			ServiceName: svc,
			LogLevel:    strings.ToUpper(level),
			Message:     truncateText(msg, 400),
			Labels:      labels,
			DeepLinkURL: deepLink,
		})
	}
	return out
}

func (s *serviceImpl) executeAndVerifyAction(
	ctx context.Context,
	inv *entities.IncidentInvestigation,
	action *entities.RemediationAction,
	cfg *entities.AgentConfig,
	stepIdx *int,
) {
	execNow := time.Now().UTC()
	action.ExecutedAt = &execNow

	execSummary, execErr := s.runRemediationAction(ctx, inv, action, cfg)
	if execErr != nil {
		action.Status = "FAILED"
		msg := fmt.Sprintf("Execution failed: %v", execErr)
		action.VerificationSummary = &msg
		_ = s.db.WithContext(ctx).Save(action).Error
		_ = s.db.WithContext(ctx).Create(&entities.InvestigationStep{
			InvestigationID: inv.ID,
			StepIndex:       *stepIdx,
			StepType:        "REMEDIATION",
			ServerName:      "opsway-agent",
			ToolName:        "remediation.execute",
			InputJSON:       action.PayloadJSON,
			OutputSummary:   msg,
			Status:          "FAILED",
			DurationMs:      time.Since(execNow).Milliseconds(),
		}).Error
		*stepIdx++
		return
	}

	// Post-fix verification using real MCP Gateway calls.
	verifyStart := time.Now()
	var verifyParts []string
	verifyParts = append(verifyParts, execSummary)

	verified := true
	if s.mcpGateway != nil {
		if logRes, err := s.mcpGateway.CallTool(ctx, inv.TeamID, "logs.tail_recent_logs", map[string]interface{}{
			"service_name": inv.ServiceName,
			"limit":        20,
		}); err == nil && logRes != nil {
			verifyParts = append(verifyParts, "Recent logs: "+truncateText(extractFirstText(logRes), 250))
		}
		if inv.MonitorID != nil {
			if chkRes, err := s.mcpGateway.CallTool(ctx, inv.TeamID, "monitoring.trigger_monitor_recheck", map[string]interface{}{
				"monitor_id": *inv.MonitorID,
			}); err == nil && chkRes != nil {
				chkText := extractFirstText(chkRes)
				verifyParts = append(verifyParts, "Monitor recheck: "+truncateText(chkText, 250))
				var chkObj map[string]interface{}
				if json.Unmarshal([]byte(chkText), &chkObj) == nil {
					if v, ok := chkObj["verified"].(bool); ok && !v {
						verified = false
					}
				}
			}
		}
	}

	if verified {
		action.Status = "VERIFIED"
	} else {
		action.Status = "EXECUTED"
	}
	verifySummary := strings.Join(verifyParts, " | ")
	action.VerificationSummary = &verifySummary
	_ = s.db.WithContext(ctx).Save(action).Error

	_ = s.db.WithContext(ctx).Create(&entities.InvestigationStep{
		InvestigationID: inv.ID,
		StepIndex:       *stepIdx,
		StepType:        "VERIFICATION",
		ServerName:      "opsway-agent",
		ToolName:        "remediation.execute_and_verify",
		InputJSON:       action.PayloadJSON,
		OutputSummary:   truncateText(verifySummary, maxStepOutputSummary),
		Status:          "COMPLETED",
		DurationMs:      time.Since(verifyStart).Milliseconds(),
	}).Error
	*stepIdx++
}

func (s *serviceImpl) runRemediationAction(
	ctx context.Context,
	inv *entities.IncidentInvestigation,
	action *entities.RemediationAction,
	cfg *entities.AgentConfig,
) (string, error) {
	// Resolve target URL / headers / payload from matching playbook or action PayloadJSON.
	var payloadMap map[string]interface{}
	_ = json.Unmarshal([]byte(action.PayloadJSON), &payloadMap)

	var targetURL, method, payloadTemplate string
	var headers map[string]string

	if pbID, ok := payloadMap["playbook_id"].(string); ok && pbID != "" {
		for _, pb := range cfg.RemediationPlaybooks {
			if pb.ID == pbID {
				targetURL = pb.TargetURL
				method = pb.Method
				headers = pb.Headers
				payloadTemplate = pb.PayloadTemplate
				break
			}
		}
	}
	if targetURL == "" {
		if u, ok := payloadMap["target_url"].(string); ok && u != "" {
			targetURL = u
		} else if u, ok := payloadMap["url"].(string); ok && u != "" {
			targetURL = u
		}
	}
	if method == "" {
		if m, ok := payloadMap["method"].(string); ok && m != "" {
			method = m
		}
	}

	// Non-webhook advisory actions (CLI_COMMAND, GITHUB_PR without webhook) are marked approved/recorded without fake execution.
	if strings.TrimSpace(targetURL) == "" {
		return fmt.Sprintf("Approved advisory action `%s` (%s); no automated webhook endpoint configured.", action.Title, action.ActionType), nil
	}

	bodyStr := payloadTemplate
	if strings.TrimSpace(bodyStr) == "" {
		defaultBody, _ := json.Marshal(map[string]interface{}{
			"investigation_id": inv.UUID,
			"incident_id":      inv.IncidentID,
			"team_id":          inv.TeamID,
			"service":          inv.ServiceName,
			"action_type":      action.ActionType,
			"action_title":     action.Title,
		})
		bodyStr = string(defaultBody)
	} else {
		bodyStr = strings.ReplaceAll(bodyStr, "{{service}}", inv.ServiceName)
		bodyStr = strings.ReplaceAll(bodyStr, "{{incident_id}}", fmt.Sprintf("%d", inv.IncidentID))
		bodyStr = strings.ReplaceAll(bodyStr, "{{investigation_id}}", inv.UUID)
	}

	req, err := http.NewRequestWithContext(ctx, defaultHTTPMethod(method), targetURL, strings.NewReader(bodyStr))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if cfg != nil && cfg.ByoaWebhookSecret != nil && *cfg.ByoaWebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(*cfg.ByoaWebhookSecret))
		mac.Write([]byte(bodyStr))
		req.Header.Set("X-Opsway-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("playbook endpoint %s returned HTTP %d: %s", targetURL, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return fmt.Sprintf("Executed %s %s (HTTP %d).", defaultHTTPMethod(method), targetURL, resp.StatusCode), nil
}

func (s *serviceImpl) HandleByoaCallback(
	ctx context.Context,
	teamID uint,
	incidentID uint,
	invUUID string,
	payload ByoaCallbackPayload,
) (*entities.IncidentInvestigation, error) {
	var inv entities.IncidentInvestigation
	if err := s.db.WithContext(ctx).Where("team_id = ? AND incident_id = ? AND uuid = ?", teamID, incidentID, invUUID).First(&inv).Error; err != nil {
		return nil, ErrNotFound
	}

	if payload.RootCauseSummary != "" {
		inv.RootCauseSummary = &payload.RootCauseSummary
	}
	if payload.DetailedDiagnosis != "" {
		inv.DetailedDiagnosis = &payload.DetailedDiagnosis
	}
	if payload.ConfidenceScore > 0 {
		inv.ConfidenceScore = &payload.ConfidenceScore
	}
	if len(payload.CorrelatedLogs) > 0 {
		inv.CorrelatedLogs = payload.CorrelatedLogs
	}
	if payload.Status != "" {
		inv.Status = entities.InvestigationStatus(payload.Status)
	} else {
		inv.Status = entities.InvestigationStatusCompleted
	}
	now := time.Now().UTC()
	inv.CompletedAt = &now
	inv.DurationMs = now.Sub(inv.StartedAt).Milliseconds()
	_ = s.db.WithContext(ctx).Save(&inv).Error

	for i, step := range payload.Steps {
		step.ID = 0
		step.InvestigationID = inv.ID
		step.StepIndex = i + 10
		_ = s.db.WithContext(ctx).Create(&step).Error
	}

	for _, action := range payload.RemediationActions {
		action.ID = 0
		action.InvestigationID = inv.ID
		action.TeamID = teamID
		action.IncidentID = incidentID
		_ = s.db.WithContext(ctx).Create(&action).Error
	}

	var updated entities.IncidentInvestigation
	_ = s.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_index asc")
		}).
		Preload("RemediationActions").
		First(&updated, inv.ID).Error

	return &updated, nil
}

func (s *serviceImpl) ApproveRemediationAction(
	ctx context.Context,
	teamID uint,
	investigationID uint,
	actionID uint,
	userID *uint,
) (*entities.IncidentInvestigation, error) {
	var action entities.RemediationAction
	if err := s.db.WithContext(ctx).Where("id = ? AND investigation_id = ? AND team_id = ?", actionID, investigationID, teamID).First(&action).Error; err != nil {
		return nil, ErrNotFound
	}

	var inv entities.IncidentInvestigation
	if err := s.db.WithContext(ctx).First(&inv, investigationID).Error; err != nil {
		return nil, ErrNotFound
	}

	cfg, _ := s.GetAgentConfig(ctx, teamID)
	action.ApprovedByUserID = userID

	var stepCount int64
	s.db.WithContext(ctx).Model(&entities.InvestigationStep{}).Where("investigation_id = ?", inv.ID).Count(&stepCount)
	nextStep := int(stepCount) + 1

	s.executeAndVerifyAction(ctx, &inv, &action, cfg, &nextStep)

	if action.Status != "FAILED" {
		inv.Status = entities.InvestigationStatusCompleted
		_ = s.db.WithContext(ctx).Save(&inv).Error
	}

	var updated entities.IncidentInvestigation
	_ = s.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_index asc")
		}).
		Preload("RemediationActions").
		First(&updated, inv.ID).Error

	return &updated, nil
}

func (s *serviceImpl) RejectRemediationAction(
	ctx context.Context,
	teamID uint,
	investigationID uint,
	actionID uint,
) (*entities.IncidentInvestigation, error) {
	var action entities.RemediationAction
	if err := s.db.WithContext(ctx).Where("id = ? AND investigation_id = ? AND team_id = ?", actionID, investigationID, teamID).First(&action).Error; err != nil {
		return nil, ErrNotFound
	}
	action.Status = "REJECTED"
	_ = s.db.WithContext(ctx).Save(&action).Error

	var updated entities.IncidentInvestigation
	_ = s.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_index asc")
		}).
		Preload("RemediationActions").
		First(&updated, investigationID).Error

	return &updated, nil
}

func parseSynthesisResult(raw string, serviceName string) agentSynthesisResult {
	cleaned := strings.TrimSpace(raw)
	// Strip markdown code fences if present
	if strings.HasPrefix(cleaned, "```") {
		if idx := strings.Index(cleaned, "\n"); idx != -1 {
			cleaned = cleaned[idx+1:]
		}
		if end := strings.LastIndex(cleaned, "```"); end != -1 {
			cleaned = cleaned[:end]
		}
		cleaned = strings.TrimSpace(cleaned)
	}
	// Extract outermost JSON object if surrounded by prose
	if start := strings.Index(cleaned, "{"); start != -1 {
		if end := strings.LastIndex(cleaned, "}"); end > start {
			candidate := cleaned[start : end+1]
			var res agentSynthesisResult
			if err := json.Unmarshal([]byte(candidate), &res); err == nil && strings.TrimSpace(res.RootCauseSummary) != "" {
				if strings.TrimSpace(res.DetailedDiagnosis) == "" {
					res.DetailedDiagnosis = res.RootCauseSummary
				}
				return res
			}
		}
	}

	summary := cleaned
	if summary == "" {
		summary = fmt.Sprintf("Investigation completed for `%s`, but the model returned an empty diagnosis.", serviceName)
	}
	return agentSynthesisResult{
		RootCauseSummary:  truncateText(summary, 500),
		DetailedDiagnosis: summary,
		ConfidenceScore:   0.65,
	}
}

func parseToolArguments(raw string) map[string]interface{} {
	args := map[string]interface{}{}
	if strings.TrimSpace(raw) == "" {
		return args
	}
	_ = json.Unmarshal([]byte(raw), &args)
	return args
}

func inferServerName(toolName string) string {
	switch {
	case strings.HasPrefix(toolName, "monitoring."):
		return "monitoring-mcp"
	case strings.HasPrefix(toolName, "logs."):
		return "log-search-mcp"
	case strings.HasPrefix(toolName, "pantomath."):
		return "pantomath-mcp"
	case strings.HasPrefix(toolName, "ext."):
		parts := strings.SplitN(toolName, ".", 3)
		if len(parts) >= 2 {
			return parts[1]
		}
		return "external-mcp"
	default:
		return "opsway-mcp"
	}
}

func sanitizeIdentifier(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "server"
	}
	return b.String()
}

func isExecutableActionType(t string) bool {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "WEBHOOK_CALL", "GITHUB_WORKFLOW", "K8S_ROLLOUT_RESTART":
		return true
	default:
		return false
	}
}

func defaultHTTPMethod(m string) string {
	m = strings.ToUpper(strings.TrimSpace(m))
	if m == "" {
		return http.MethodPost
	}
	return m
}

func firstNonEmptyString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func appendUniqueLogs(dst []entities.CorrelatedLogEntry, src []entities.CorrelatedLogEntry, limit int) []entities.CorrelatedLogEntry {
	seen := make(map[string]bool, len(dst))
	for _, e := range dst {
		seen[e.LogID+"|"+e.Message] = true
	}
	for _, e := range src {
		if len(dst) >= limit {
			break
		}
		key := e.LogID + "|" + e.Message
		if seen[key] {
			continue
		}
		seen[key] = true
		dst = append(dst, e)
	}
	return dst
}

func extractFirstText(res *mcp.ToolCallResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	return res.Content[0].Text
}

func truncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
