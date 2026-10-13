package alerting

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type AlertRuleResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Condition string    `json:"condition"`
	Channels  string    `json:"channels"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newAlertRuleResponse(r *entities.AlertRule) AlertRuleResponse {
	return AlertRuleResponse{
		ID:        r.ID,
		Name:      r.Name,
		Condition: r.Condition,
		Channels:  r.Channels,
		Enabled:   r.Enabled,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func ruleError(c hs.AuthenticatedContext, err error, msg string) error {
	if errors.Is(err, alerting.ErrNotFound) {
		return echo.ErrNotFound
	}

	c.Log.WithError(err).Error(msg)

	return echo.ErrInternalServerError
}

type GetAlertRulesRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) GetAlertRules(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetAlertRulesRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	rules, err := h.AlertingService.GetRulesByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		return ruleError(c, err, "failed to get alert rules")
	}

	resp := make([]AlertRuleResponse, len(*rules))
	for i := range *rules {
		resp[i] = newAlertRuleResponse(&(*rules)[i])
	}

	return c.JSON(http.StatusOK, resp)
}

type GetAlertRuleRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
	RuleID uint `param:"ruleId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) GetAlertRule(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetAlertRuleRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	rule, err := h.AlertingService.GetRuleByTeamIDAndID(c.Request().Context(), req.TeamID, req.RuleID)
	if err != nil {
		return ruleError(c, err, "failed to get alert rule")
	}

	return c.JSON(http.StatusOK, newAlertRuleResponse(rule))
}

type PostAlertRuleRequest struct {
	TeamID    uint   `param:"teamId" validate:"required,numeric,gt=0"`
	Name      string `json:"name" validate:"required,max=255"`
	Condition string `json:"condition" validate:"required,max=255"`
	Channels  string `json:"channels" validate:"max=1024"`
	Enabled   bool   `json:"enabled"`
}

func (h *Handlers) PostAlertRule(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostAlertRuleRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostAlertRuleRequest")

		return echo.ErrBadRequest
	}

	rule := entities.AlertRule{
		TeamID:    req.TeamID,
		Name:      req.Name,
		Condition: req.Condition,
		Channels:  req.Channels,
		Enabled:   req.Enabled,
	}

	if err := h.AlertingService.CreateRule(c.Request().Context(), &rule); err != nil {
		return ruleError(c, err, "failed to create alert rule")
	}

	return c.JSON(http.StatusCreated, newAlertRuleResponse(&rule))
}

type PutAlertRuleRequest struct {
	TeamID    uint   `param:"teamId" validate:"required,numeric,gt=0"`
	RuleID    uint   `param:"ruleId" validate:"required,numeric,gt=0"`
	Name      string `json:"name" validate:"required,max=255"`
	Condition string `json:"condition" validate:"required,max=255"`
	Channels  string `json:"channels" validate:"max=1024"`
	Enabled   bool   `json:"enabled"`
}

func (h *Handlers) PutAlertRule(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutAlertRuleRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutAlertRuleRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	rule := entities.AlertRule{
		ID:        req.RuleID,
		TeamID:    req.TeamID,
		Name:      req.Name,
		Condition: req.Condition,
		Channels:  req.Channels,
		Enabled:   req.Enabled,
	}

	if err := h.AlertingService.UpdateRule(ctx, &rule); err != nil {
		return ruleError(c, err, "failed to update alert rule")
	}

	updated, err := h.AlertingService.GetRuleByTeamIDAndID(ctx, req.TeamID, req.RuleID)
	if err != nil {
		return ruleError(c, err, "failed to get alert rule")
	}

	return c.JSON(http.StatusOK, newAlertRuleResponse(updated))
}

type DeleteAlertRuleRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
	RuleID uint `param:"ruleId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) DeleteAlertRule(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteAlertRuleRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	if err := h.AlertingService.DeleteRule(c.Request().Context(), req.TeamID, req.RuleID); err != nil {
		return ruleError(c, err, "failed to delete alert rule")
	}

	return c.NoContent(http.StatusNoContent)
}

type GetAlertRuleTriggersRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
	RuleID uint `param:"ruleId" validate:"required,numeric,gt=0"`
	Offset *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit  *int `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetAlertRuleTriggersResponse struct {
	TotalCount int64                                 `json:"totalCount"`
	Triggers   []GetAlertRuleTriggersResponseTrigger `json:"triggers"`
}

type GetAlertRuleTriggersResponseTrigger struct {
	ID         uint      `json:"id"`
	IncidentID *uint     `json:"incidentId,omitempty"`
	Channels   string    `json:"channels"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (h *Handlers) GetAlertRuleTriggers(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetAlertRuleTriggersRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	if _, err := h.AlertingService.GetRuleByTeamIDAndID(ctx, req.TeamID, req.RuleID); err != nil {
		return ruleError(c, err, "failed to get alert rule")
	}

	triggers, total, err := h.AlertingService.GetTriggersByRuleID(ctx, req.TeamID, req.RuleID, req.Offset, req.Limit)
	if err != nil {
		return ruleError(c, err, "failed to get alert rule triggers")
	}

	resp := GetAlertRuleTriggersResponse{
		TotalCount: total,
		Triggers:   make([]GetAlertRuleTriggersResponseTrigger, len(*triggers)),
	}
	for i, t := range *triggers {
		resp.Triggers[i] = GetAlertRuleTriggersResponseTrigger{
			ID:         t.ID,
			IncidentID: t.IncidentID,
			Channels:   t.Channels,
			CreatedAt:  t.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, resp)
}
