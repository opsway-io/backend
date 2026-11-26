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
	PantomathURL         string
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
	pantomathURL string,
	productGuard func(entities.ProductSlug) func(echo.HandlerFunc) echo.HandlerFunc,
) {
	if internalAgentKey == "" {
		internalAgentKey = "valid_agent_key"
	}
	if pantomathURL == "" {
		pantomathURL = "http://pantomath.pantomath.svc.cluster.local:8000"
	}
	h := &Handlers{
		InvestigationService: investigationService,
		MCPGateway:           mcpGateway,
		ApiKeyService:        apiKeyService,
		UsageService:         usageService,
		UsageGuard:           usageGuard,
		InternalAgentKey:     internalAgentKey,
		PantomathURL:         pantomathURL,
		Logger:               logger,
	}

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	// Unified MCP Gateway endpoint (supports API Key or Bearer Token)
	root.POST("/mcp", h.PostMCPGateway)
	root.GET("/mcp/tools", h.GetMCPTools)

	// BYOA External Agent Callback (authenticated via investigation UUID & team/incident)
	root.POST("/teams/:teamId/incidents/:incidentId/investigations/:uuid/callback", h.PostByoaCallback)

	// Attach productGuard as route-level middleware instead of teamGroup.Group("", ...)
	// because Echo's empty-prefix Group registers NotFoundHandler on /v1/teams/:teamId,
	// overwriting GET/PUT/DELETE /v1/teams/:teamId.
	withProduct := func(p entities.ProductSlug) []echo.MiddlewareFunc {
		if productGuard == nil {
			return nil
		}
		return []echo.MiddlewareFunc{productGuard(p)}
	}
	pantomathMW := withProduct(entities.ProductPantomath)
	agentMW := withProduct(entities.ProductManagedAgent)
	logSearchMW := withProduct(entities.ProductLogSearch)

	// Modular Product Entitlements & Usage Monitoring
	teamGroup.GET("/entitlements", AuthHandler(h.GetEntitlements))
	teamGroup.PUT("/entitlements", AuthHandler(h.PutEntitlement))
	teamGroup.GET("/usage", AuthHandler(h.GetTeamUsage))
	teamGroup.PUT("/overage-settings", AuthHandler(h.PutOverageSettings))
	teamGroup.PUT("/pantomath/active-product", AuthHandler(h.PutActivePantomathProduct), pantomathMW...)
	teamGroup.POST("/pantomath/query", AuthHandler(h.PostPantomathQuery), pantomathMW...)
	teamGroup.POST("/pantomath/products", AuthHandler(h.PostCreatePantomathProduct), pantomathMW...)
	teamGroup.POST("/pantomath/products/:product/repos", AuthHandler(h.PostAddPantomathProductRepos), pantomathMW...)
	teamGroup.GET("/pantomath/products/:product/repos", AuthHandler(h.GetPantomathProductRepos), pantomathMW...)

	// Cross-Product Service Catalog
	teamGroup.GET("/service-catalog", AuthHandler(h.GetServiceCatalog))
	teamGroup.POST("/service-catalog", AuthHandler(h.PostServiceCatalogItem))
	teamGroup.PUT("/service-catalog/:serviceId", AuthHandler(h.PutServiceCatalogItem))
	teamGroup.DELETE("/service-catalog/:serviceId", AuthHandler(h.DeleteServiceCatalogItem))

	// Autonomous Agent Configuration & Multi-Product Trigger Rules
	teamGroup.GET("/agent-config", AuthHandler(h.GetAgentConfig), agentMW...)
	teamGroup.PUT("/agent-config", AuthHandler(h.PutAgentConfig), agentMW...)
	teamGroup.GET("/agent-triggers", AuthHandler(h.GetAgentTriggers), agentMW...)
	teamGroup.POST("/agent-triggers", AuthHandler(h.PostAgentTrigger), agentMW...)
	teamGroup.PUT("/agent-triggers/:ruleId", AuthHandler(h.PutAgentTrigger), agentMW...)
	teamGroup.DELETE("/agent-triggers/:ruleId", AuthHandler(h.DeleteAgentTrigger), agentMW...)
	teamGroup.POST("/agent-events", AuthHandler(h.PostTeamAgentEvent), agentMW...)

	// Internal Product Hooks (Log-Search Evaluator, Pantomath Deploy Webhooks & Quota Enforcement)
	root.GET("/internal/agent-triggers", h.GetInternalAgentTriggers)
	root.POST("/internal/agent-events", h.PostInternalAgentEvent)
	root.GET("/internal/quota-state", h.GetInternalQuotaState)

	// Incident Investigations & Remediation Actions
	teamGroup.GET("/investigations", AuthHandler(h.GetTeamInvestigations), agentMW...)
	teamGroup.GET("/incidents/:incidentId/investigations", AuthHandler(h.GetIncidentInvestigations), agentMW...)
	teamGroup.POST("/incidents/:incidentId/investigations", AuthHandler(h.PostTriggerInvestigation), agentMW...)
	teamGroup.POST("/investigations/:investigationId/remediations/:actionId/approve", AuthHandler(h.PostApproveRemediation), agentMW...)
	teamGroup.POST("/investigations/:investigationId/remediations/:actionId/reject", AuthHandler(h.PostRejectRemediation), agentMW...)

	// Log-Search Federation Proxy for Dashboard / Service Auto-Discovery
	teamGroup.GET("/logs/services", AuthHandler(h.GetLogSearchServices), logSearchMW...)
}
