package suite

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/investigation"
	"github.com/opsway-io/backend/internal/mcp"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type mcpRateWindow struct {
	minute int64
	count  int
}

var (
	mcpRateMu      sync.Mutex
	mcpRateWindows = map[uint]*mcpRateWindow{}
)

func allowMCPRate(teamID uint, limitPerMin int) bool {
	if limitPerMin <= 0 {
		return true
	}
	nowMin := time.Now().UTC().Unix() / 60
	mcpRateMu.Lock()
	defer mcpRateMu.Unlock()
	w, ok := mcpRateWindows[teamID]
	if !ok || w.minute != nowMin {
		mcpRateWindows[teamID] = &mcpRateWindow{minute: nowMin, count: 1}
		return true
	}
	if w.count >= limitPerMin {
		return false
	}
	w.count++
	return true
}

func (h *Handlers) PostMCPGateway(c echo.Context) error {
	var rpcReq mcp.JSONRPCRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&rpcReq); err != nil {
		return c.JSON(http.StatusBadRequest, mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &mcp.JSONRPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		})
	}

	teamID := uint(1)
	if tHeader := c.Request().Header.Get("X-Opsway-Team-ID"); tHeader != "" {
		if parsed, err := strconv.ParseUint(tHeader, 10, 64); err == nil && parsed > 0 {
			teamID = uint(parsed)
		}
	} else if authHeader := c.Request().Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		rawKey := strings.TrimPrefix(authHeader, "Bearer ")
		if h.ApiKeyService != nil {
			if apiKey, err := h.ApiKeyService.GetByPlaintext(c.Request().Context(), rawKey); err == nil && apiKey != nil {
				teamID = apiKey.TeamID
			}
		}
	}

	// Enforce per-tier MCP Gateway rate limit (60 Free / 300 Team / 1000 Enterprise req/min)
	mcpLimit := 60
	if h.UsageGuard != nil {
		mcpLimit = h.UsageGuard.GetTeamMCPRateLimit(c.Request().Context(), teamID)
	}
	if !allowMCPRate(teamID, mcpLimit) {
		return c.JSON(http.StatusTooManyRequests, mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      rpcReq.ID,
			Error: &mcp.JSONRPCError{
				Code:    -32029,
				Message: "MCP Gateway rate limit exceeded for your plan. Upgrade in Opsway Settings -> Billing for higher throughput.",
			},
		})
	}

	resp := h.MCPGateway.HandleJSONRPC(c.Request().Context(), teamID, rpcReq)
	if h.UsageService != nil {
		if err := h.UsageService.IncrementCounter(c.Request().Context(), teamID, entities.ProductManagedAgent, "mcp_calls", 1); err != nil {
			h.Logger.WithError(err).Warn("failed to record MCP usage counter")
		}
		if rpcReq.Method == "tools/call" && strings.Contains(string(rpcReq.Params), "pantomath") {
			_ = h.UsageService.IncrementCounter(c.Request().Context(), teamID, entities.ProductPantomath, "mcp_calls", 1)
		}
	}
	if rpcReq.ID == nil {
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) GetMCPTools(c echo.Context) error {
	teamID := uint(1)
	if q := c.QueryParam("teamId"); q != "" {
		if parsed, err := strconv.ParseUint(q, 10, 64); err == nil && parsed > 0 {
			teamID = uint(parsed)
		}
	}
	tools := h.MCPGateway.ListTools(c.Request().Context(), teamID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"protocolVersion": mcp.ProtocolVersion,
		"gatewayUrl":      "https://api.opsway.eu/v1/mcp",
		"logSearchMcpUrl": "https://logs.opsway.eu/mcp",
		"tools":           tools,
	})
}

type TeamParamRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetEntitlements(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[TeamParamRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	entitlements, err := h.InvestigationService.GetEntitlements(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get team entitlements")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"entitlements": entitlements,
	})
}

type PutEntitlementRequest struct {
	TeamID  uint                     `param:"teamId" validate:"required,numeric,gte=0"`
	Product entities.ProductSlug     `json:"product" validate:"required"`
	Tier    entities.EntitlementTier `json:"tier" validate:"required"`
	Enabled bool                     `json:"enabled"`
}

func (h *Handlers) PutEntitlement(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutEntitlementRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	ent, err := h.InvestigationService.UpdateEntitlement(c.Request().Context(), req.TeamID, req.Product, req.Tier, req.Enabled)
	if err != nil {
		c.Log.WithError(err).Error("failed to update entitlement")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, ent)
}

func (h *Handlers) GetServiceCatalog(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[TeamParamRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	items, err := h.InvestigationService.ListServiceCatalog(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to list service catalog")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"services": items,
	})
}

type UpsertServiceCatalogRequest struct {
	TeamID            uint              `param:"teamId" validate:"required,numeric,gte=0"`
	ServiceID         uint              `param:"serviceId"`
	Slug              string            `json:"slug" validate:"required"`
	Name              string            `json:"name" validate:"required"`
	Description       string            `json:"description"`
	LogSearchServices []string          `json:"logSearchServices"`
	MonitorIDs        []uint            `json:"monitorIds"`
	RepositoryURL     *string           `json:"repositoryUrl"`
	PantomathProduct  *string           `json:"pantomathProduct"`
	EnvironmentLabels map[string]string `json:"environmentLabels"`
}

func (h *Handlers) PostServiceCatalogItem(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[UpsertServiceCatalogRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	item := &entities.ServiceCatalogItem{
		TeamID:            req.TeamID,
		Slug:              req.Slug,
		Name:              req.Name,
		Description:       req.Description,
		LogSearchServices: req.LogSearchServices,
		MonitorIDs:        req.MonitorIDs,
		RepositoryURL:     req.RepositoryURL,
		PantomathProduct:  req.PantomathProduct,
		EnvironmentLabels: req.EnvironmentLabels,
	}
	created, err := h.InvestigationService.UpsertServiceCatalogItem(c.Request().Context(), item)
	if err != nil {
		c.Log.WithError(err).Error("failed to create service catalog item")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusCreated, created)
}

func (h *Handlers) PutServiceCatalogItem(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[UpsertServiceCatalogRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	item := &entities.ServiceCatalogItem{
		ID:                req.ServiceID,
		TeamID:            req.TeamID,
		Slug:              req.Slug,
		Name:              req.Name,
		Description:       req.Description,
		LogSearchServices: req.LogSearchServices,
		MonitorIDs:        req.MonitorIDs,
		RepositoryURL:     req.RepositoryURL,
		PantomathProduct:  req.PantomathProduct,
		EnvironmentLabels: req.EnvironmentLabels,
	}
	updated, err := h.InvestigationService.UpsertServiceCatalogItem(c.Request().Context(), item)
	if err != nil {
		c.Log.WithError(err).Error("failed to update service catalog item")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, updated)
}

type DeleteServiceCatalogRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	ServiceID uint `param:"serviceId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteServiceCatalogItem(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteServiceCatalogRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.InvestigationService.DeleteServiceCatalogItem(c.Request().Context(), req.TeamID, req.ServiceID); err != nil {
		return echo.ErrInternalServerError
	}
	return c.NoContent(http.StatusNoContent)
}

func maskAgentSecrets(cfg *entities.AgentConfig) *entities.AgentConfig {
	if cfg == nil {
		return nil
	}
	out := *cfg
	if out.ByoaWebhookSecret != nil && *out.ByoaWebhookSecret != "" {
		masked := "********"
		out.ByoaWebhookSecret = &masked
	}
	if out.CustomLLMAPIKey != nil && *out.CustomLLMAPIKey != "" {
		masked := "********"
		out.CustomLLMAPIKey = &masked
	}
	return &out
}

func (h *Handlers) GetAgentConfig(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[TeamParamRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	cfg, err := h.InvestigationService.GetAgentConfig(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get agent config")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, maskAgentSecrets(cfg))
}

type PutAgentConfigRequest struct {
	TeamID                 uint                                 `param:"teamId" validate:"required,numeric,gte=0"`
	Enabled                *bool                                `json:"enabled"`
	ExecutionMode          entities.AgentExecutionMode          `json:"executionMode"`
	AutoTriggerOnConsensus bool                                 `json:"autoTriggerOnConsensus"`
	AutoTriggerOnAnomaly   bool                                 `json:"autoTriggerOnAnomaly"`
	AutoTriggerOnLogSpike  bool                                 `json:"autoTriggerOnLogSpike"`
	MinOccurrences         int                                  `json:"minOccurrences"`
	CooldownMinutes        int                                  `json:"cooldownMinutes"`
	RemediationLevel       entities.RemediationAutonomyLevel    `json:"remediationLevel"`
	ByoaWebhookURL         *string                              `json:"byoaWebhookUrl"`
	ByoaWebhookSecret      *string                              `json:"byoaWebhookSecret"`
	CustomLLMProvider      *string                              `json:"customLlmProvider"`
	CustomLLMBaseURL       *string                              `json:"customLlmBaseUrl"`
	CustomLLMModel         *string                              `json:"customLlmModel"`
	CustomLLMAPIKey        *string                              `json:"customLlmApiKey"`
	ExternalMCPServers     []entities.ExternalMCPServerConfig   `json:"externalMcpServers"`
	RemediationPlaybooks   []entities.RemediationPlaybookConfig `json:"remediationPlaybooks"`
}

func (h *Handlers) PutAgentConfig(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutAgentConfigRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	existing, _ := h.InvestigationService.GetAgentConfig(c.Request().Context(), req.TeamID)
	enabled := true
	if existing != nil {
		enabled = existing.Enabled
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	cfg := &entities.AgentConfig{
		TeamID:                 req.TeamID,
		Enabled:                enabled,
		ExecutionMode:          req.ExecutionMode,
		AutoTriggerOnConsensus: req.AutoTriggerOnConsensus,
		AutoTriggerOnAnomaly:   req.AutoTriggerOnAnomaly,
		AutoTriggerOnLogSpike:  req.AutoTriggerOnLogSpike,
		MinOccurrences:         req.MinOccurrences,
		CooldownMinutes:        req.CooldownMinutes,
		RemediationLevel:       req.RemediationLevel,
		ByoaWebhookURL:         req.ByoaWebhookURL,
		ByoaWebhookSecret:      req.ByoaWebhookSecret,
		CustomLLMProvider:      req.CustomLLMProvider,
		CustomLLMBaseURL:       req.CustomLLMBaseURL,
		CustomLLMModel:         req.CustomLLMModel,
		CustomLLMAPIKey:        req.CustomLLMAPIKey,
		ExternalMCPServers:     req.ExternalMCPServers,
		RemediationPlaybooks:   req.RemediationPlaybooks,
	}
	updated, err := h.InvestigationService.UpdateAgentConfig(c.Request().Context(), req.TeamID, cfg)
	if err != nil {
		c.Log.WithError(err).Error("failed to update agent config")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, maskAgentSecrets(updated))
}

func (h *Handlers) GetTeamInvestigations(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[TeamParamRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	list, err := h.InvestigationService.ListInvestigationsByTeam(c.Request().Context(), req.TeamID, 30)
	if err != nil {
		c.Log.WithError(err).Error("failed to list investigations")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"investigations": list,
	})
}

type IncidentParamRequest struct {
	TeamID     uint `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint `param:"incidentId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetIncidentInvestigations(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[IncidentParamRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	list, err := h.InvestigationService.GetInvestigationsByIncident(c.Request().Context(), req.TeamID, req.IncidentID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident investigations")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"investigations": list,
	})
}

type PostTriggerInvestigationRequest struct {
	TeamID        uint    `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID    uint    `param:"incidentId" validate:"required,numeric,gte=0"`
	UserPrompt    *string `json:"userPrompt"`
	TriggerSource string  `json:"triggerSource"`
}

func (h *Handlers) PostTriggerInvestigation(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostTriggerInvestigationRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	source := req.TriggerSource
	if source == "" {
		source = "MANUAL_UI"
	}
	inv, err := h.InvestigationService.TriggerInvestigation(c.Request().Context(), investigation.TriggerRequest{
		TeamID:        req.TeamID,
		IncidentID:    req.IncidentID,
		TriggerSource: source,
		UserPrompt:    req.UserPrompt,
		Force:         true,
	})
	if err != nil {
		if errors.Is(err, investigation.ErrQuotaExceeded) {
			return echo.NewHTTPError(http.StatusPaymentRequired, err.Error())
		}
		if errors.Is(err, investigation.ErrNotFound) {
			return echo.ErrNotFound
		}
		c.Log.WithError(err).Error("failed to trigger investigation")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusCreated, inv)
}

type ByoaCallbackRequest struct {
	TeamID     uint   `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint   `param:"incidentId" validate:"required,numeric,gte=0"`
	UUID       string `param:"uuid" validate:"required"`
}

func (h *Handlers) PostByoaCallback(c echo.Context) error {
	req, err := helpers.Bind[ByoaCallbackRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	var payload investigation.ByoaCallbackPayload
	if err := json.NewDecoder(c.Request().Body).Decode(&payload); err != nil {
		return echo.ErrBadRequest
	}
	inv, err := h.InvestigationService.HandleByoaCallback(c.Request().Context(), req.TeamID, req.IncidentID, req.UUID, payload)
	if err != nil {
		return echo.ErrNotFound
	}
	return c.JSON(http.StatusOK, inv)
}

type RemediationActionRequest struct {
	TeamID          uint `param:"teamId" validate:"required,numeric,gte=0"`
	InvestigationID uint `param:"investigationId" validate:"required,numeric,gte=0"`
	ActionID        uint `param:"actionId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) PostApproveRemediation(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[RemediationActionRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	userID := c.UserID
	inv, err := h.InvestigationService.ApproveRemediationAction(c.Request().Context(), req.TeamID, req.InvestigationID, req.ActionID, &userID)
	if err != nil {
		if errors.Is(err, investigation.ErrNotFound) {
			return echo.ErrNotFound
		}
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, inv)
}

func (h *Handlers) PostRejectRemediation(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[RemediationActionRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	inv, err := h.InvestigationService.RejectRemediationAction(c.Request().Context(), req.TeamID, req.InvestigationID, req.ActionID)
	if err != nil {
		if errors.Is(err, investigation.ErrNotFound) {
			return echo.ErrNotFound
		}
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, inv)
}

func (h *Handlers) GetLogSearchServices(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[TeamParamRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	res, _ := h.MCPGateway.CallTool(c.Request().Context(), req.TeamID, "logs.list_services", map[string]interface{}{})
	services := []string{"checkout-service", "backend-api", "payment-worker", "auth-service", "status-page"}
	if res != nil && len(res.Content) > 0 {
		var parsed struct {
			Services []string `json:"services"`
		}
		if err := json.Unmarshal([]byte(res.Content[0].Text), &parsed); err == nil && len(parsed.Services) > 0 {
			services = parsed.Services
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"services": services,
	})
}

func parseUsageTime(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func (h *Handlers) GetTeamUsage(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[TeamParamRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	if h.UsageService == nil {
		return echo.ErrServiceUnavailable
	}

	now := time.Now().UTC()
	var from, to time.Time

	switch c.QueryParam("range") {
	case "7d":
		from = now.AddDate(0, 0, -6)
		to = now
	case "30d":
		from = now.AddDate(0, 0, -29)
		to = now
	case "90d":
		from = now.AddDate(0, 0, -89)
		to = now
	default:
		// current_cycle or explicit from/to
		if parsedFrom, ok := parseUsageTime(c.QueryParam("from")); ok {
			from = parsedFrom
		}
		if parsedTo, ok := parseUsageTime(c.QueryParam("to")); ok {
			to = parsedTo
		}
	}

	res, err := h.UsageService.GetTeamUsage(c.Request().Context(), req.TeamID, from, to)
	if err != nil {
		c.Log.WithError(err).Error("failed to get team usage")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, res)
}

type PutOverageSettingsRequest struct {
	TeamID               uint `param:"teamId" validate:"required,numeric,gte=0"`
	AllowOverage         bool `json:"allowOverage"`
	OverageSpendCapCents *int `json:"overageSpendCapCents"`
}

func (h *Handlers) PutOverageSettings(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutOverageSettingsRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	if h.UsageService == nil {
		return echo.ErrServiceUnavailable
	}
	settings, err := h.UsageService.UpdateOverageSettings(c.Request().Context(), req.TeamID, req.AllowOverage, req.OverageSpendCapCents)
	if err != nil {
		if strings.Contains(err.Error(), "require a Team or Enterprise plan") {
			return c.JSON(http.StatusPaymentRequired, map[string]interface{}{
				"code":       "UPGRADE_REQUIRED",
				"message":    err.Error(),
				"upgradeUrl": "/settings/billing",
			})
		}
		c.Log.WithError(err).Error("failed to update overage settings")
		return echo.ErrInternalServerError
	}
	if h.UsageGuard != nil {
		go h.UsageGuard.EvaluateTeam(c.Request().Context(), req.TeamID)
	}
	return c.JSON(http.StatusOK, settings)
}

func (h *Handlers) GetInternalQuotaState(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	expected := "Bearer " + h.InternalAgentKey
	if h.InternalAgentKey != "" && authHeader != expected {
		return echo.ErrUnauthorized
	}

	tenant := strings.TrimSpace(c.QueryParam("tenant"))
	product := entities.ProductSlug(strings.TrimSpace(c.QueryParam("product")))
	if product == "" {
		product = entities.ProductLogSearch
	}

	teamID := uint(1)
	if strings.HasPrefix(tenant, "team_") {
		if parsed, err := strconv.ParseUint(strings.TrimPrefix(tenant, "team_"), 10, 64); err == nil && parsed > 0 {
			teamID = uint(parsed)
		}
	} else if parsed, err := strconv.ParseUint(tenant, 10, 64); err == nil && parsed > 0 {
		teamID = uint(parsed)
	}

	state := entities.QuotaStateNormal
	if h.UsageGuard != nil {
		state = h.UsageGuard.GetProductEnforcementState(c.Request().Context(), teamID, product)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"teamId":  teamID,
		"tenant":  tenant,
		"product": product,
		"state":   state,
	})
}
