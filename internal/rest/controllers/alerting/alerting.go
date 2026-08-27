package alerting

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	TeamService     team.Service
	AlertingService alerting.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	alertingService alerting.Service,
) {
	h := &Handlers{
		TeamService:     teamService,
		AlertingService: alertingService,
	}

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	cg := e.Group("/teams/:teamId/alerting")

	cg.GET("", AuthHandler(h.GetAlertRules))
	cg.POST("", AuthHandler(h.PostAlertRule))
	cg.GET("/:ruleId", AuthHandler(h.GetAlertRule))
	cg.PUT("/:ruleId", AuthHandler(h.PutAlertRule))
	cg.DELETE("/:ruleId", AuthHandler(h.DeleteAlertRule))
	cg.GET("/:ruleId/triggers", AuthHandler(h.GetAlertRuleTriggers))
}

type GetAlertRulesRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetAlertRules(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetAlertRulesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetAlertRulesRequest")
		return echo.ErrBadRequest
	}

	rules, err := h.AlertingService.GetAllByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get alert rules")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, rules)
}

type GetAlertRuleRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
	RuleID uint `param:"ruleId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetAlertRule(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetAlertRuleRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetAlertRuleRequest")
		return echo.ErrBadRequest
	}

	rule, err := h.AlertingService.GetByIDAndTeamID(c.Request().Context(), req.TeamID, req.RuleID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get alert rule")
		return echo.ErrNotFound
	}

	return c.JSON(http.StatusOK, rule)
}

type PostAlertRuleRequest struct {
	TeamID    uint   `param:"teamId" validate:"required,numeric,gte=0"`
	Name      string `json:"name" validate:"required"`
	Condition string `json:"condition" validate:"required"`
	Channels  string `json:"channels" validate:"required"`
	Enabled   bool   `json:"enabled"`
}

func (h *Handlers) PostAlertRule(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PostAlertRuleRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostAlertRuleRequest")
		return echo.ErrBadRequest
	}

	rule := &entities.AlertRule{
		TeamID:    req.TeamID,
		Name:      req.Name,
		Condition: req.Condition,
		Channels:  req.Channels,
		Enabled:   req.Enabled,
	}

	err = h.AlertingService.Create(c.Request().Context(), rule)
	if err != nil {
		c.Log.WithError(err).Error("failed to create alert rule")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, rule)
}

type PutAlertRuleRequest struct {
	TeamID    uint   `param:"teamId" validate:"required,numeric,gte=0"`
	RuleID    uint   `param:"ruleId" validate:"required,numeric,gte=0"`
	Name      string `json:"name" validate:"required"`
	Condition string `json:"condition" validate:"required"`
	Channels  string `json:"channels" validate:"required"`
	Enabled   bool   `json:"enabled"`
}

func (h *Handlers) PutAlertRule(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PutAlertRuleRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutAlertRuleRequest")
		return echo.ErrBadRequest
	}

	rule, err := h.AlertingService.GetByIDAndTeamID(c.Request().Context(), req.TeamID, req.RuleID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get alert rule for update")
		return echo.ErrNotFound
	}

	rule.Name = req.Name
	rule.Condition = req.Condition
	rule.Channels = req.Channels
	rule.Enabled = req.Enabled

	err = h.AlertingService.Update(c.Request().Context(), rule)
	if err != nil {
		c.Log.WithError(err).Error("failed to update alert rule")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, rule)
}

type DeleteAlertRuleRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
	RuleID uint `param:"ruleId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteAlertRule(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteAlertRuleRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteAlertRuleRequest")
		return echo.ErrBadRequest
	}

	rule, err := h.AlertingService.GetByIDAndTeamID(c.Request().Context(), req.TeamID, req.RuleID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get alert rule for deletion")
		return echo.ErrNotFound
	}

	err = h.AlertingService.Delete(c.Request().Context(), rule)
	if err != nil {
		c.Log.WithError(err).Error("failed to delete alert rule")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
