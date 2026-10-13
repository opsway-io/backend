package metrics

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/apikey"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	checkService check.Service,
	apiKeyService apikey.Service,
	db *gorm.DB,
) {
	h := &Handlers{
		CheckService: checkService,
		DB:           db,
	}

	ApiKeyGuard := middleware.ApiKeyGuardFactory(logger, apiKeyService)

	metricsGroup := e.Group(
		"/teams/:teamId/metrics",
		ApiKeyGuard(),
	)

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	metricsGroup.GET("/prometheus", AuthHandler(h.GetPrometheusMetrics))
}
