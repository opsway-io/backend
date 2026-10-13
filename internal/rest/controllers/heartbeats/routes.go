package heartbeats

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opsway-io/backend/internal/heartbeat"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	HeartbeatService heartbeat.Service
	Logger           *logrus.Entry
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	heartbeatService heartbeat.Service,
) {
	h := &Handlers{
		HeartbeatService: heartbeatService,
		Logger:           logger,
	}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)
	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	group := e.Group(
		"/teams/:teamId/heartbeats",
		TeamGuard(),
	)

	group.GET("", AuthHandler(h.GetHeartbeats))
	group.POST("", AuthHandler(h.PostHeartbeat), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	group.GET("/:heartbeatId", AuthHandler(h.GetHeartbeat))
	group.PUT("/:heartbeatId", AuthHandler(h.PutHeartbeat), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	group.DELETE("/:heartbeatId", AuthHandler(h.DeleteHeartbeat), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

}

// RegisterPublic registers the unauthenticated ping endpoint shown on the heartbeat page.
func RegisterPublic(
	e *echo.Group,
	logger *logrus.Entry,
	heartbeatService heartbeat.Service,
) {
	h := &Handlers{
		HeartbeatService: heartbeatService,
		Logger:           logger,
	}

	limiter := middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate: 2, Burst: 30, ExpiresIn: 3 * time.Minute,
		}),
	})

	e.POST("/teams/:teamId/heartbeats/:heartbeatId/ping", h.PingHeartbeat, limiter)
	e.GET("/teams/:teamId/heartbeats/:heartbeatId/ping", h.PingHeartbeat, limiter)
}
