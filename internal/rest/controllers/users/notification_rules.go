package users

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type NotificationRuleResponse struct {
	ID      uint   `json:"id"`
	Channel string `json:"channel"`
	Delay   int    `json:"delay"`
}

type GetNotificationRulesResponse struct {
	Rules []NotificationRuleResponse `json:"rules"`
}

func (h *Handlers) GetNotificationRules(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()
	rules, err := h.UserService.GetNotificationRules(ctx, c.UserID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get notification rules")
		return echo.ErrInternalServerError
	}

	resp := GetNotificationRulesResponse{Rules: make([]NotificationRuleResponse, len(rules))}
	for i, r := range rules {
		resp.Rules[i] = NotificationRuleResponse{
			ID:      r.ID,
			Channel: string(r.Channel),
			Delay:   r.Delay,
		}
	}

	return c.JSON(http.StatusOK, resp)
}

type PutNotificationRulesRequest struct {
	Rules []NotificationRuleRequest `json:"rules" validate:"required"`
}

type NotificationRuleRequest struct {
	Channel string `json:"channel" validate:"required,oneof=email sms slack webhook voice"`
	Delay   int    `json:"delay" validate:"min=0"`
}

func (h *Handlers) PutNotificationRules(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutNotificationRulesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutNotificationRulesRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	var rules []entities.UserNotificationRule
	for _, r := range req.Rules {
		rules = append(rules, entities.UserNotificationRule{
			Channel: entities.ChannelType(r.Channel),
			Delay:   r.Delay,
		})
	}

	if err := h.UserService.SetNotificationRules(ctx, c.UserID, rules); err != nil {
		c.Log.WithError(err).Error("failed to set notification rules")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
