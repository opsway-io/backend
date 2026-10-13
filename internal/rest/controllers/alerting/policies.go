package alerting

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

/*
	Escalation policy
*/

type OnCallRotation struct {
	UserID uint `json:"userId" validate:"required,gt=0"`
	Tier   uint `json:"tier" validate:"required,gte=1,lte=10"`
}

type EscalationPolicyResponse struct {
	Name                     string           `json:"name"`
	EscalationTimeoutMinutes uint             `json:"escalationTimeoutMinutes"`
	EnforcedChannel          *string          `json:"enforcedChannel,omitempty"`
	Rotations                []OnCallRotation `json:"rotations"`
}

type GetEscalationPolicyRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) GetEscalationPolicy(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetEscalationPolicyRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	policy, err := h.PolicyService.GetEscalationPolicy(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get escalation policy")

		return echo.ErrInternalServerError
	}

	resp := EscalationPolicyResponse{
		Name:                     policy.Name,
		EscalationTimeoutMinutes: policy.EscalationTimeoutMinutes,
		EnforcedChannel:          policy.EnforcedChannel,
		Rotations:                make([]OnCallRotation, len(policy.Rotations)),
	}
	for i, r := range policy.Rotations {
		resp.Rotations[i] = OnCallRotation{UserID: r.UserID, Tier: r.Tier}
	}

	return c.JSON(http.StatusOK, resp)
}

type PutEscalationPolicyRequest struct {
	TeamID                   uint             `param:"teamId" validate:"required,numeric,gt=0"`
	Name                     string           `json:"name" validate:"required,max=255"`
	EscalationTimeoutMinutes uint             `json:"escalationTimeoutMinutes" validate:"gte=0,lte=1440"`
	EnforcedChannel          string           `json:"enforcedChannel" validate:"omitempty,oneof=email sms voice"`
	Rotations                []OnCallRotation `json:"rotations" validate:"dive"`
}

func (h *Handlers) PutEscalationPolicy(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutEscalationPolicyRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutEscalationPolicyRequest")

		return echo.ErrBadRequest
	}

	var enforced *string
	if req.EnforcedChannel != "" {
		enforced = &req.EnforcedChannel
	}

	policy := entities.EscalationPolicy{
		TeamID:                   req.TeamID,
		Name:                     req.Name,
		EscalationTimeoutMinutes: req.EscalationTimeoutMinutes,
		EnforcedChannel:          enforced,
		Rotations:                make([]entities.EscalationRotation, len(req.Rotations)),
	}
	for i, r := range req.Rotations {
		policy.Rotations[i] = entities.EscalationRotation{UserID: r.UserID, Tier: r.Tier}
	}

	if err := h.PolicyService.PutEscalationPolicy(c.Request().Context(), &policy); err != nil {
		if errors.Is(err, alerting.ErrRotationUserNotOnTeam) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		c.Log.WithError(err).Error("failed to update escalation policy")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

/*
	User notification rules
*/

type NotificationRule struct {
	ID      uint   `json:"id,omitempty"`
	Channel string `json:"channel" validate:"required,oneof=email sms voice"`
	Delay   uint   `json:"delay" validate:"gte=0,lte=1440"`
}

type GetNotificationRulesRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
}

type GetNotificationRulesResponse struct {
	Rules []NotificationRule `json:"rules"`
}

func (h *Handlers) GetNotificationRules(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetNotificationRulesRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	rules, err := h.PolicyService.GetNotificationRules(c.Request().Context(), req.UserID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get notification rules")

		return echo.ErrInternalServerError
	}

	resp := GetNotificationRulesResponse{Rules: make([]NotificationRule, len(*rules))}
	for i, r := range *rules {
		resp.Rules[i] = NotificationRule{ID: r.ID, Channel: r.Channel, Delay: r.Delay}
	}

	return c.JSON(http.StatusOK, resp)
}

type PutNotificationRulesRequest struct {
	UserID uint               `param:"userId" validate:"required,numeric,gt=0"`
	Rules  []NotificationRule `json:"rules" validate:"max=50,dive"`
}

func (h *Handlers) PutNotificationRules(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutNotificationRulesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutNotificationRulesRequest")

		return echo.ErrBadRequest
	}

	rules := make([]entities.UserNotificationRule, len(req.Rules))
	for i, r := range req.Rules {
		rules[i] = entities.UserNotificationRule{Channel: r.Channel, Delay: r.Delay}
	}

	if err := h.PolicyService.PutNotificationRules(c.Request().Context(), req.UserID, rules); err != nil {
		c.Log.WithError(err).Error("failed to update notification rules")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
