package teams

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/billing"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	AuthenticationService authentication.Service
	TeamService           team.Service
	UserService           user.Service
	BillingService        billing.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	userService user.Service,
	billingService billing.Service,
) {
	h := &Handlers{
		TeamService:    teamService,
		UserService:    userService,
		BillingService: billingService,
	}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	e.POST("/teams", AuthHandler(h.PostTeam))
	e.POST("/teams/available", AuthHandler(h.PostTeamAvailable))
	e.POST("/teams/invites/accept", AuthHandler(h.PostTeamInvitesAccept))
	teamsGroup := e.Group(
		"/teams/:teamId",
		TeamGuard(),
	)

	teamsGroup.GET("", AuthHandler(h.GetTeam))
	teamsGroup.PUT("", AuthHandler(h.PutTeam), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	teamsGroup.DELETE("", AuthHandler(h.DeleteTeam), AllowedRoles(mw.UserRoleOwner))

	teamsGroup.GET("/users", AuthHandler(h.GetTeamUsers))
	teamsGroup.DELETE("/users/:userId", AuthHandler(h.DeleteTeamUser), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	teamsGroup.PUT("/users/:userId", AuthHandler(h.PutTeamUser), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	teamsGroup.POST("/users/invites", AuthHandler(h.PostTeamUsersInvites), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	teamsGroup.PUT("/avatar", AuthHandler(h.PutTeamAvatar), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	teamsGroup.DELETE("/avatar", AuthHandler(h.DeleteTeamAvatar), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	teamsGroup.GET("/config", AuthHandler(h.PostConfig), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	teamsGroup.POST("/create-checkout-session", AuthHandler(h.PostCreateCheckoutSession), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	teamsGroup.GET("/checkout-session", AuthHandler(h.GetCheckoutSession), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	teamsGroup.POST("/customer-portal", AuthHandler(h.PostCustomerPortal), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	teamsGroup.GET("/customer-session", AuthHandler(h.GetCustomerSession), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	teamsGroup.GET("/products", AuthHandler(h.GetProducts), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
}
