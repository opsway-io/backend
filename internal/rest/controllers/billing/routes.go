package billing

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	TeamService           team.Service
	UserService           user.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	userService user.Service,
) {
	h := &Handlers{
		TeamService: teamService,
		UserService: userService,
	}

	StripeHandler := handlers.StripeHandlerFactory(logger)

	e.POST("/webhook", StripeHandler(h.handleWebhook))

}
