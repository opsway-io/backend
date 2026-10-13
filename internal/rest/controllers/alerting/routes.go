package alerting

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/rest/handlers"
	mw "github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	TeamService     team.Service
	AlertingService alerting.Service
	PolicyService   alerting.PolicyService
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	alertingService alerting.Service,
	policyService alerting.PolicyService,
) {
	h := &Handlers{
		TeamService:     teamService,
		AlertingService: alertingService,
		PolicyService:   policyService,
	}

	TeamGuard := mw.TeamGuardFactory(logger, teamService)
	AllowedRoles := mw.RoleGuardFactory(logger, teamService)
	CurrentUserGuard := mw.CurrentUserGuardFactory(logger)
	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	// Alert rules

	rules := e.Group("/teams/:teamId/alerting", TeamGuard())

	rules.GET("", AuthHandler(h.GetAlertRules))
	rules.POST("", AuthHandler(h.PostAlertRule), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	rules.GET("/:ruleId", AuthHandler(h.GetAlertRule))
	rules.PUT("/:ruleId", AuthHandler(h.PutAlertRule), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	rules.DELETE("/:ruleId", AuthHandler(h.DeleteAlertRule), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))
	rules.GET("/:ruleId/triggers", AuthHandler(h.GetAlertRuleTriggers))

	// Escalation policy

	escalation := e.Group("/teams/:teamId/escalation", TeamGuard())

	escalation.GET("", AuthHandler(h.GetEscalationPolicy))
	escalation.PUT("", AuthHandler(h.PutEscalationPolicy), AllowedRoles(mw.UserRoleOwner, mw.UserRoleAdmin))

	// User notification rules

	userRules := e.Group("/users/:userId/notification-rules", CurrentUserGuard())

	userRules.GET("", AuthHandler(h.GetNotificationRules))
	userRules.PUT("", AuthHandler(h.PutNotificationRules))
}
