package rest

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	auth "github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/billing"
	"github.com/opsway-io/backend/internal/changelog"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/escalation"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/heartbeats"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/maintenance"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/report"
	"github.com/opsway-io/backend/internal/rest/controllers"
	"github.com/opsway-io/backend/internal/rest/controllers/authentication"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/apikey"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Config struct {
	Port                 uint32 `default:"8001"`
	GZIPCompressionLevel int    `default:"5"`
}

type Server struct {
	echo   *echo.Echo
	config Config
}

func NewServer(
	conf Config,
	oauthConfig *authentication.OAuthConfig,
	authConfig *auth.Config,
	logger *logrus.Logger,
	authenticationService auth.Service,
	userService user.Service,
	teamService team.Service,
	alertingService alerting.Service,
	monitorService monitor.Service,
	checkService check.Service,
	billingService billing.Service,
	changelogService changelog.Service,
	heartbeatService heartbeats.Service,
	incidentService incident.Service,
	maintenanceService maintenance.Service,
	reportsService report.Service,
	statusPageService statuspage.Service,
	escalationService escalation.Service,
	eventService event.Service,
	apiKeyService apikey.Service,
	emailSender email.Sender,
	availableLocations []string,
	db *gorm.DB,
	ch *gorm.DB,
	statusPageBaseURL string,
) (*Server, error) {
	cookieService := helpers.NewCookieService(authConfig)

	e := echo.New()

	e.HideBanner = true
	e.HidePort = true
	e.Validator = helpers.NewValidator()

	e.Use(
		middleware.Recover(),
		middleware.Logger(),
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{"http://localhost:5173", "http://localhost:5174", "https://app.opsway.eu", "https://opsway.eu"},
			AllowCredentials: true,
		}),
		middleware.SecureWithConfig(middleware.SecureConfig{
			XSSProtection:         "1; mode=block",
			ContentTypeNosniff:    "nosniff",
			XFrameOptions:         "DENY",
			HSTSMaxAge:            31536000,
			ContentSecurityPolicy: "default-src 'self'",
		}),
		middleware.BodyLimit("2M"),
		middleware.GzipWithConfig(middleware.GzipConfig{
			Level:   conf.GZIPCompressionLevel,
			Skipper: middleware.DefaultGzipConfig.Skipper,
		}),
	)

	controllers.Register(
		e,
		logger.WithField("module", "rest_controllers"),
		oauthConfig,
		authConfig,
		cookieService,
		authenticationService,
		userService,
		teamService,
		alertingService,
		monitorService,
		checkService,
		billingService,
		changelogService,
		heartbeatService,
		incidentService,
		maintenanceService,
		reportsService,
		statusPageService,
		escalationService,
		eventService,
		apiKeyService,
		emailSender,
		availableLocations,
		db,
		ch,
		statusPageBaseURL,
	)

	return &Server{
		echo:   e,
		config: conf,
	}, nil
}

func (s *Server) Start() error {
	return errors.Wrap(s.echo.Start(fmt.Sprintf(":%d", s.config.Port)), "Failed to start server")
}

func (s *Server) Shutdown(ctx context.Context) error {
	return errors.Wrap(s.echo.Shutdown(ctx), "Failed to shutdown server")
}
