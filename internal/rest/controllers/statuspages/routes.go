package statuspages

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	StatusPageService statuspage.Service
	Logger            *logrus.Entry
}

// Register registers the team scoped (authenticated) status page routes.
func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	statusPageService statuspage.Service,
) {
	h := &Handlers{StatusPageService: statusPageService, Logger: logger}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)
	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	group := e.Group(
		"/teams/:teamId/status-pages",
		TeamGuard(),
	)

	group.GET("", AuthHandler(h.GetStatusPages))
	group.POST("", AuthHandler(h.PostStatusPage), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	group.GET("/:statusPageId", AuthHandler(h.GetStatusPage))
	group.PUT("/:statusPageId", AuthHandler(h.PutStatusPage), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	group.DELETE("/:statusPageId", AuthHandler(h.DeleteStatusPage), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
}

// RegisterPublic registers the unauthenticated, rate limited public status page routes.
func RegisterPublic(
	e *echo.Group,
	logger *logrus.Entry,
	statusPageService statuspage.Service,
) {
	h := &Handlers{StatusPageService: statusPageService, Logger: logger}

	// Generous limit for page views / badges / widgets (per client IP).
	readLimiter := middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate: 5, Burst: 60, ExpiresIn: 3 * time.Minute,
		}),
	})

	// Strict limit for subscription attempts (per client IP).
	subscribeLimiter := middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate: 0.1, Burst: 5, ExpiresIn: 10 * time.Minute,
		}),
	})

	group := e.Group("/public/status-pages/:domain")

	group.GET("", h.GetPublicStatusPage, readLimiter)
	group.GET("/badge.svg", h.GetBadge, readLimiter)
	group.GET("/widget.js", h.GetWidget, readLimiter)
	group.POST("/subscribe", h.PostSubscribe, subscribeLimiter)
	group.GET("/subscribe/verify/:token", h.GetVerifySubscriber, readLimiter)
}
