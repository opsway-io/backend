package suite

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/apikey"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/investigation"
	"github.com/opsway-io/backend/internal/mcp"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/usage"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	InvestigationService investigation.Service
	MCPGateway           mcp.Gateway
	ApiKeyService        apikey.Service
	UsageService         usage.Service
	UsageGuard           *usage.Guard
	InternalAgentKey     string
	Logger               *logrus.Entry
}

func Register(
	root *echo.Group,
	teamGroup *echo.Group,
	logger *logrus.Entry,
	investigationService investigation.Service,
	mcpGateway mcp.Gateway,
	apiKeyService apikey.Service,
	usageService usage.Service,
	usageGuard *usage.Guard,
	internalAgentKey string,
	productGuard func(entities.ProductSlug) func(echo.HandlerFunc) echo.HandlerFunc,
) {
	if internalAgentKey == "" {
		internalAgentKey = "valid_agent_key"
	}
	h := &Handlers{
		InvestigationService: investigationService,
		MCPGateway:           mcpGateway,
		ApiKeyService:        apiKeyService,
		UsageService:         usageService,
		UsageGuard:           usageGuard,
		InternalAgentKey:     internalAgentKey,
		Logger:               logger,
	}

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	// Unified MCP Gateway endpoint (supports API Key or Bearer Token)
	root.POST("/mcp", h.PostMCPGateway)
	root.GET("/mcp/tools", h.GetMCPTools)

	// BYOA External Agent Callback (authenticated via investigation UUID & team/incident)
	root.POST("/teams/:teamId/incidents/:incidentId/investigations/:uuid/callback", h.PostByoaCallback)

	pantomathGroup := teamGroup
	agentGroup := teamGroup
	logSearchGroup := teamGroup
	if productGuard != nil {
		pantomathGroup = teamGroup.Group("", productGuard(entities.ProductPantomath))
		agentGroup = teamGroup.Group("", productGuard(entities.ProductManagedAgent))
		logSearchGroup = teamGroup.Group("", productGuard(entities.ProductLogSearch))
	}

	// Modular Product Entitlements & Usage Monitoring
	teamGroup.GET("/entitlements", AuthHandler(h.GetEntitlements))
	teamGroup.PUT("/entitlements", AuthHandler(h.PutEntitlement))
	teamGroup.GET("/usage", AuthHandler(h.GetTeamUsage))
	teamGroup.PUT("/overage-settings", AuthHandler(h.PutOverageSettings))
	pantomathGroup.PUT("/pantomath/active-product", AuthHandler(h.PutActivePantomathProduct))

	// Cross-Product Service Catalog
	teamGroup.GET("/service-catalog", AuthHandler(h.GetServiceCatalog))
	teamGroup.POST("/service-catalog", AuthHandler(h.PostServiceCatalogItem))
	teamGroup.PUT("/service-catalog/:serviceId", AuthHandler(h.PutServiceCatalogItem))
	teamGroup.DELETE("/service-catalog/:serviceId", AuthHandler(h.DeleteServiceCatalogItem))

	// Autonomous Agent Configuration & Multi-Product Trigger Rules
	agentGroup.GET("/agent-config", AuthHandler(h.GetAgentConfig))
	agentGroup.PUT("/agent-config", AuthHandler(h.PutAgentConfig))
	agentGroup.GET("/agent-triggers", AuthHandler(h.GetAgentTriggers))
	agentGroup.POST("/agent-triggers", AuthHandler(h.PostAgentTrigger))
	agentGroup.PUT("/agent-triggers/:ruleId", AuthHandler(h.PutAgentTrigger))
	agentGroup.DELETE("/agent-triggers/:ruleId", AuthHandler(h.DeleteAgentTrigger))
	agentGroup.POST("/agent-events", AuthHandler(h.PostTeamAgentEvent))

	// Internal Product Hooks (Log-Search Evaluator, Pantomath Deploy Webhooks & Quota Enforcement)
	root.GET("/internal/agent-triggers", h.GetInternalAgentTriggers)
	root.POST("/internal/agent-events", h.PostInternalAgentEvent)
	root.GET("/internal/quota-state", h.GetInternalQuotaState)

	// Incident Investigations & Remediation Actions
	agentGroup.GET("/investigations", AuthHandler(h.GetTeamInvestigations))
	agentGroup.GET("/incidents/:incidentId/investigations", AuthHandler(h.GetIncidentInvestigations))
	agentGroup.POST("/incidents/:incidentId/investigations", AuthHandler(h.PostTriggerInvestigation))
	agentGroup.POST("/investigations/:investigationId/remediations/:actionId/approve", AuthHandler(h.PostApproveRemediation))
	agentGroup.POST("/investigations/:investigationId/remediations/:actionId/reject", AuthHandler(h.PostRejectRemediation))

	// Log-Search Federation Proxy for Dashboard / Service Auto-Discovery
	logSearchGroup.GET("/logs/services", AuthHandler(h.GetLogSearchServices))
}
