package webhooks

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/billing"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	BillingService        billing.Service
	TeamService           team.Service
	IncidentService       incident.Service
	EmailSender           email.Sender
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	billingService billing.Service,
	teamService team.Service,
	incidentService incident.Service,
	emailSender email.Sender,
) {
	h := &Handlers{
		BillingService:  billingService,
		TeamService:     teamService,
		IncidentService: incidentService,
		EmailSender:     emailSender,
	}

	root := e.Group(
		"/webhooks",
	)

	// Stripe

	StripeGuard := middleware.StripeGuardFactory(logger)
	StripeHandler := handlers.StripeHandlerFactory(logger)

	root.POST("/stripe", StripeHandler(h.handleWebhook), StripeGuard())

	// Slack
	root.POST("/slack/interactive", h.PostSlackInteractive)

	// APM Integrations
	root.POST("/datadog/:teamId", h.PostDatadogWebhook)
	root.POST("/new_relic/:teamId", h.PostNewRelicWebhook)
}
