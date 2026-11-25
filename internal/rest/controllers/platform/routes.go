package platform

import (
	"github.com/labstack/echo/v4"
	platformSvc "github.com/opsway-io/backend/internal/platform"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	PlatformService platformSvc.Service
	Logger          *logrus.Entry
}

func Register(
	authRoot *echo.Group,
	logger *logrus.Entry,
	platformService platformSvc.Service,
) {
	h := &Handlers{
		PlatformService: platformService,
		Logger:          logger,
	}

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)
	PlatformOwnerGuard := middleware.PlatformOwnerGuardFactory(logger, platformService)

	group := authRoot.Group("/platform")
	group.GET("/products", AuthHandler(h.GetUserProducts))

	admin := group.Group("/admin", PlatformOwnerGuard())
	admin.GET("/products", AuthHandler(h.GetAdminProductFlags))
	admin.PUT("/products/:product", AuthHandler(h.PutAdminProductFlag))
	admin.GET("/owners", AuthHandler(h.GetPlatformOwners))
	admin.POST("/owners", AuthHandler(h.PostPlatformOwner))
	admin.DELETE("/owners/:userId", AuthHandler(h.DeletePlatformOwner))
}
