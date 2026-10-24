package suite

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/apikey"
	"github.com/opsway-io/backend/internal/investigation"
	"github.com/opsway-io/backend/internal/mcp"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	InvestigationService investigation.Service
	MCPGateway           mcp.Gateway
	ApiKeyService        apikey.Service
	Logger               *logrus.Entry
}

func Register(
	root *echo.Group,
	teamGroup *echo.Group,
	logger *logrus.Entry,
	investigationService investigation.Service,
	mcpGateway mcp.Gateway,
	apiKeyService apikey.Service,
) {
	h := &Handlers{
		InvestigationService: investigationService,
		MCPGateway:           mcpGateway,
		ApiKeyService:        apiKeyService,
		Logger:               logger,
	}

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	// Unified MCP Gateway endpoint (supports API Key or Bearer Token)
	root.POST("/mcp", h.PostMCPGateway)
	root.GET("/mcp/tools", h.GetMCPTools)

	// BYOA External Agent Callback (authenticated via investigation UUID & team/incident)
	root.POST("/teams/:teamId/incidents/:incidentId/investigations/:uuid/callback", h.PostByoaCallback)

	// Modular Product Entitlements
	teamGroup.GET("/entitlements", AuthHandler(h.GetEntitlements))
	teamGroup.PUT("/entitlements", AuthHandler(h.PutEntitlement))

	// Cross-Product Service Catalog
	teamGroup.GET("/service-catalog", AuthHandler(h.GetServiceCatalog))
	teamGroup.POST("/service-catalog", AuthHandler(h.PostServiceCatalogItem))
	teamGroup.PUT("/service-catalog/:serviceId", AuthHandler(h.PutServiceCatalogItem))
	teamGroup.DELETE("/service-catalog/:serviceId", AuthHandler(h.DeleteServiceCatalogItem))

	// Autonomous Agent Configuration (Managed vs BYOA)
	teamGroup.GET("/agent-config", AuthHandler(h.GetAgentConfig))
	teamGroup.PUT("/agent-config", AuthHandler(h.PutAgentConfig))

	// Incident Investigations & Remediation Actions
	teamGroup.GET("/investigations", AuthHandler(h.GetTeamInvestigations))
	teamGroup.GET("/incidents/:incidentId/investigations", AuthHandler(h.GetIncidentInvestigations))
	teamGroup.POST("/incidents/:incidentId/investigations", AuthHandler(h.PostTriggerInvestigation))
	teamGroup.POST("/investigations/:investigationId/remediations/:actionId/approve", AuthHandler(h.PostApproveRemediation))
	teamGroup.POST("/investigations/:investigationId/remediations/:actionId/reject", AuthHandler(h.PostRejectRemediation))

	// Log-Search Federation Proxy for Dashboard / Service Auto-Discovery
	teamGroup.GET("/logs/services", AuthHandler(h.GetLogSearchServices))
}
