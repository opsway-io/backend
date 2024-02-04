package webhooks

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/billing"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	BillingService        billing.Service
	TeamService           team.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	billingService billing.Service,
	teamService team.Service,
) {
	h := &Handlers{
		BillingService: billingService,
	}

	root := e.Group(
		"/webhooks",
	)

	// Stripe

	StripeGuard := middleware.StripeGuardFactory(logger)
	StripeHandler := handlers.StripeHandlerFactory(logger)

	root.POST("/stripe", StripeHandler(h.handleWebhook), StripeGuard())
}
