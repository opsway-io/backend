package billings

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/billing"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	BillingService        billing.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	billingService billing.Service,
) {
	h := &Handlers{
		BillingService: billingService,
	}

	StripeHandler := handlers.StripeHandlerFactory(logger)

	e.POST("/webhook", StripeHandler(h.handleWebhook))

}
