package alerting

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
)

var ErrRotationUserNotOnTeam = errors.New("rotation user is not a member of the team")

const (
	DefaultEscalationPolicyName    = "Default"
	DefaultEscalationTimeoutMinute = 15
)

// PolicyService manages team escalation policies and user notification rules.
type PolicyService interface {
	// GetEscalationPolicy returns the team's policy, or an unsaved default policy if none exists.
	GetEscalationPolicy(ctx context.Context, teamID uint) (*entities.EscalationPolicy, error)
	PutEscalationPolicy(ctx context.Context, policy *entities.EscalationPolicy) error

	GetNotificationRules(ctx context.Context, userID uint) (*[]entities.UserNotificationRule, error)
	PutNotificationRules(ctx context.Context, userID uint, rules []entities.UserNotificationRule) error
}

type PolicyServiceImpl struct {
	repository PolicyRepository
}

func NewPolicyService(repository PolicyRepository) PolicyService {
	return &PolicyServiceImpl{repository: repository}
}

func (s *PolicyServiceImpl) GetEscalationPolicy(ctx context.Context, teamID uint) (*entities.EscalationPolicy, error) {
	policy, err := s.repository.GetEscalationPolicy(ctx, teamID)
	if errors.Is(err, ErrNotFound) {
		return &entities.EscalationPolicy{
			TeamID:                   teamID,
			Name:                     DefaultEscalationPolicyName,
			EscalationTimeoutMinutes: DefaultEscalationTimeoutMinute,
			Rotations:                []entities.EscalationRotation{},
		}, nil
	}

	return policy, err
}

func (s *PolicyServiceImpl) PutEscalationPolicy(ctx context.Context, policy *entities.EscalationPolicy) error {
	if len(policy.Rotations) > 0 {
		unique := map[uint]bool{}
		ids := []uint{}
		for _, r := range policy.Rotations {
			if !unique[r.UserID] {
				unique[r.UserID] = true
				ids = append(ids, r.UserID)
			}
		}

		count, err := s.repository.CountTeamMembers(ctx, policy.TeamID, ids)
		if err != nil {
			return err
		}
		if int(count) != len(ids) {
			return ErrRotationUserNotOnTeam
		}
	}

	return s.repository.ReplaceEscalationPolicy(ctx, policy)
}

func (s *PolicyServiceImpl) GetNotificationRules(ctx context.Context, userID uint) (*[]entities.UserNotificationRule, error) {
	return s.repository.GetNotificationRules(ctx, userID)
}

func (s *PolicyServiceImpl) PutNotificationRules(ctx context.Context, userID uint, rules []entities.UserNotificationRule) error {
	return s.repository.ReplaceNotificationRules(ctx, userID, rules)
}
