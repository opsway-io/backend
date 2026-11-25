package suite

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/investigation"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type AgentTriggerRulePayload struct {
	TeamID                   uint                              `param:"teamId" validate:"required,numeric,gte=0"`
	RuleID                   uint                              `param:"ruleId" validate:"omitempty,numeric,gte=0"`
	Product                  entities.ProductSlug              `json:"product"`
	TriggerType              string                            `json:"triggerType"`
	Name                     string                            `json:"name"`
	Description              string                            `json:"description"`
	Enabled                  bool                              `json:"enabled"`
	ServiceFilter            string                            `json:"serviceFilter"`
	Threshold                float64                           `json:"threshold"`
	WindowSeconds            int                               `json:"windowSeconds"`
	Pattern                  string                            `json:"pattern"`
	CooldownMinutes          int                               `json:"cooldownMinutes"`
	CreateIncident           bool                              `json:"createIncident"`
	RemediationLevelOverride entities.RemediationAutonomyLevel `json:"remediationLevelOverride"`
}

type DeleteTriggerRuleRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
	RuleID uint `param:"ruleId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetAgentTriggers(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[TeamParamRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	rules, err := h.InvestigationService.ListTriggerRules(c.Request().Context(), req.TeamID)
	if err != nil {
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"triggers": rules,
	})
}

func (h *Handlers) PostAgentTrigger(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[AgentTriggerRulePayload](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	if req.Product == "" || req.TriggerType == "" || req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "product, triggerType, and name are required")
	}
	rule, err := h.InvestigationService.CreateTriggerRule(c.Request().Context(), &entities.AgentTriggerRule{
		TeamID:                   req.TeamID,
		Product:                  req.Product,
		TriggerType:              req.TriggerType,
		Name:                     req.Name,
		Description:              req.Description,
		Enabled:                  req.Enabled,
		ServiceFilter:            req.ServiceFilter,
		Threshold:                req.Threshold,
		WindowSeconds:            req.WindowSeconds,
		Pattern:                  req.Pattern,
		CooldownMinutes:          req.CooldownMinutes,
		CreateIncident:           req.CreateIncident,
		RemediationLevelOverride: req.RemediationLevelOverride,
	})
	if err != nil {
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusCreated, rule)
}

func (h *Handlers) PutAgentTrigger(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[AgentTriggerRulePayload](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	updated, err := h.InvestigationService.UpdateTriggerRule(c.Request().Context(), req.TeamID, req.RuleID, &entities.AgentTriggerRule{
		Name:                     req.Name,
		Description:              req.Description,
		Enabled:                  req.Enabled,
		ServiceFilter:            req.ServiceFilter,
		Threshold:                req.Threshold,
		WindowSeconds:            req.WindowSeconds,
		Pattern:                  req.Pattern,
		CooldownMinutes:          req.CooldownMinutes,
		CreateIncident:           req.CreateIncident,
		RemediationLevelOverride: req.RemediationLevelOverride,
	})
	if err != nil {
		if errors.Is(err, investigation.ErrNotFound) {
			return echo.ErrNotFound
		}
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, updated)
}

func (h *Handlers) DeleteAgentTrigger(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteTriggerRuleRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.InvestigationService.DeleteTriggerRule(c.Request().Context(), req.TeamID, req.RuleID); err != nil {
		return echo.ErrInternalServerError
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handlers) PostTeamAgentEvent(c hs.AuthenticatedContext) error {
	teamReq, err := helpers.Bind[TeamParamRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}
	var payload investigation.AgentEventRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&payload); err != nil {
		return echo.ErrBadRequest
	}
	payload.TeamID = teamReq.TeamID
	if payload.Product == "" {
		payload.Product = "webhook"
	}
	if payload.TriggerType == "" {
		payload.TriggerType = "webhook.custom_alert"
	}

	inv, err := h.InvestigationService.TriggerFromEvent(c.Request().Context(), payload)
	if err != nil {
		if errors.Is(err, investigation.ErrQuotaExceeded) {
			return echo.NewHTTPError(http.StatusPaymentRequired, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if inv == nil {
		return c.JSON(http.StatusAccepted, map[string]interface{}{
			"skipped": true,
			"reason":  "deduplicated or disabled by agent policy",
		})
	}
	return c.JSON(http.StatusCreated, inv)
}

func (h *Handlers) checkInternalAuth(c echo.Context) bool {
	auth := strings.TrimSpace(c.Request().Header.Get("Authorization"))
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if token != "" && token == h.InternalAgentKey {
			return true
		}
	}
	return false
}

func (h *Handlers) GetInternalAgentTriggers(c echo.Context) error {
	if !h.checkInternalAuth(c) {
		return echo.ErrUnauthorized
	}
	product := entities.ProductSlug(strings.TrimSpace(c.QueryParam("product")))
	rules, err := h.InvestigationService.ListActiveTriggerRulesByProduct(c.Request().Context(), product)
	if err != nil {
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"triggers": rules,
	})
}

func (h *Handlers) PostInternalAgentEvent(c echo.Context) error {
	if !h.checkInternalAuth(c) {
		return echo.ErrUnauthorized
	}
	var payload investigation.AgentEventRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&payload); err != nil {
		return echo.ErrBadRequest
	}
	if payload.TeamID == 0 || payload.Product == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "teamId and product are required")
	}

	inv, err := h.InvestigationService.TriggerFromEvent(c.Request().Context(), payload)
	if err != nil {
		if errors.Is(err, investigation.ErrQuotaExceeded) {
			return echo.NewHTTPError(http.StatusPaymentRequired, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if inv == nil {
		return c.JSON(http.StatusAccepted, map[string]interface{}{
			"skipped": true,
			"reason":  "deduplicated or disabled by policy",
		})
	}
	return c.JSON(http.StatusCreated, inv)
}
