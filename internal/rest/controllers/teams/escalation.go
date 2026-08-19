package teams

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/rest/helpers"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"gorm.io/gorm"
)

type EscalationPolicyRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

type OnCallRotationResponse struct {
	UserID uint `json:"userId"`
	Tier   int  `json:"tier"`
}

type EscalationPolicyResponse struct {
	Name                     string                   `json:"name"`
	EscalationTimeoutMinutes int                      `json:"escalationTimeoutMinutes"`
	Rotations                []OnCallRotationResponse `json:"rotations"`
}

func (h *Handlers) GetEscalationPolicy(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[EscalationPolicyRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind EscalationPolicyRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()
	policy, err := h.EscalationService.GetPolicyByTeamID(ctx, req.TeamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return default empty policy if none exists
			return c.JSON(http.StatusOK, EscalationPolicyResponse{
				Name: "Default Escalation",
				EscalationTimeoutMinutes: 15,
				Rotations: []OnCallRotationResponse{},
			})
		}
		c.Log.WithError(err).Error("failed to get escalation policy")
		return echo.ErrInternalServerError
	}

	rotations, err := h.EscalationService.GetRotationsByPolicyID(ctx, policy.ID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get rotations")
		return echo.ErrInternalServerError
	}

	var rotationResps []OnCallRotationResponse
	for _, r := range rotations {
		rotationResps = append(rotationResps, OnCallRotationResponse{
			UserID: r.UserID,
			Tier:   r.Tier,
		})
	}

	return c.JSON(http.StatusOK, EscalationPolicyResponse{
		Name: policy.Name,
		EscalationTimeoutMinutes: policy.EscalationTimeoutMinutes,
		Rotations: rotationResps,
	})
}

type PutEscalationPolicyRequest struct {
	TeamID                   uint                     `param:"teamId" validate:"required,numeric,gte=0"`
	Name                     string                   `json:"name" validate:"required"`
	EscalationTimeoutMinutes int                      `json:"escalationTimeoutMinutes" validate:"required,min=1"`
	Rotations                []OnCallRotationResponse `json:"rotations"`
}

func (h *Handlers) PutEscalationPolicy(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutEscalationPolicyRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutEscalationPolicyRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	policy, err := h.EscalationService.GetPolicyByTeamID(ctx, req.TeamID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.Log.WithError(err).Error("failed to get escalation policy")
		return echo.ErrInternalServerError
	}

	if errors.Is(err, gorm.ErrRecordNotFound) || policy == nil {
		policy = &entities.EscalationPolicy{
			TeamID: req.TeamID,
			Name: req.Name,
			EscalationTimeoutMinutes: req.EscalationTimeoutMinutes,
		}
		if err := h.EscalationService.CreatePolicy(ctx, policy); err != nil {
			c.Log.WithError(err).Error("failed to create escalation policy")
			return echo.ErrInternalServerError
		}
	} else {
		policy.Name = req.Name
		policy.EscalationTimeoutMinutes = req.EscalationTimeoutMinutes
		if err := h.EscalationService.UpdatePolicy(ctx, policy); err != nil {
			c.Log.WithError(err).Error("failed to update escalation policy")
			return echo.ErrInternalServerError
		}
	}

	var newRotations []entities.OnCallRotation
	for _, r := range req.Rotations {
		newRotations = append(newRotations, entities.OnCallRotation{
			EscalationPolicyID: policy.ID,
			UserID: r.UserID,
			Tier: r.Tier,
		})
	}

	if err := h.EscalationService.SetRotations(ctx, policy.ID, newRotations); err != nil {
		c.Log.WithError(err).Error("failed to set rotations")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusOK)
}
