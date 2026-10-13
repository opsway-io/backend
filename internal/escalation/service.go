package escalation

import (
	"context"
	"errors"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Service interface {
	GetPolicyByTeamID(ctx context.Context, teamID uint) (*entities.EscalationPolicy, error)
	CreatePolicy(ctx context.Context, policy *entities.EscalationPolicy) error
	UpdatePolicy(ctx context.Context, policy *entities.EscalationPolicy) error

	GetRotationsByPolicyID(ctx context.Context, policyID uint) ([]entities.OnCallRotation, error)
	GetRotationsByPolicyIDAndTier(ctx context.Context, policyID uint, tier int) ([]entities.OnCallRotation, error)
	SetRotations(ctx context.Context, policyID uint, rotations []entities.OnCallRotation) error

	// Helper to get tier 1 and tier 2 users quickly for a team
	GetOnCallUsersByTeamID(ctx context.Context, teamID uint, tier int) ([]uint, error)

	CreateEscalationJob(ctx context.Context, job *entities.EscalationJob) error
	GetPendingEscalationJobs(ctx context.Context, before time.Time) ([]entities.EscalationJob, error)
	MarkEscalationJobProcessed(ctx context.Context, id uint) error

	CreateNotificationJob(ctx context.Context, job *entities.NotificationJob) error
	GetPendingNotificationJobs(ctx context.Context, before time.Time) ([]entities.NotificationJob, error)
	MarkNotificationJobProcessed(ctx context.Context, id uint) error
}

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &ServiceImpl{repo: repo}
}

func (s *ServiceImpl) GetPolicyByTeamID(ctx context.Context, teamID uint) (*entities.EscalationPolicy, error) {
	return s.repo.GetPolicyByTeamID(ctx, teamID)
}

func (s *ServiceImpl) CreatePolicy(ctx context.Context, policy *entities.EscalationPolicy) error {
	return s.repo.CreatePolicy(ctx, policy)
}

func (s *ServiceImpl) UpdatePolicy(ctx context.Context, policy *entities.EscalationPolicy) error {
	return s.repo.UpdatePolicy(ctx, policy)
}

func (s *ServiceImpl) GetRotationsByPolicyID(ctx context.Context, policyID uint) ([]entities.OnCallRotation, error) {
	return s.repo.GetRotationsByPolicyID(ctx, policyID)
}

func (s *ServiceImpl) GetRotationsByPolicyIDAndTier(ctx context.Context, policyID uint, tier int) ([]entities.OnCallRotation, error) {
	return s.repo.GetRotationsByPolicyIDAndTier(ctx, policyID, tier)
}

func (s *ServiceImpl) SetRotations(ctx context.Context, policyID uint, rotations []entities.OnCallRotation) error {
	return s.repo.SetRotations(ctx, policyID, rotations)
}

func (s *ServiceImpl) GetOnCallUsersByTeamID(ctx context.Context, teamID uint, tier int) ([]uint, error) {
	policy, err := s.repo.GetPolicyByTeamID(ctx, teamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []uint{}, nil
		}
		return nil, err
	}

	rotations, err := s.repo.GetRotationsByPolicyIDAndTier(ctx, policy.ID, tier)
	if err != nil {
		return nil, err
	}

	var userIDs []uint
	for _, r := range rotations {
		userIDs = append(userIDs, r.UserID)
	}

	return userIDs, nil
}

func (s *ServiceImpl) CreateEscalationJob(ctx context.Context, job *entities.EscalationJob) error {
	return s.repo.CreateEscalationJob(ctx, job)
}

func (s *ServiceImpl) GetPendingEscalationJobs(ctx context.Context, before time.Time) ([]entities.EscalationJob, error) {
	return s.repo.GetPendingEscalationJobs(ctx, before)
}

func (s *ServiceImpl) MarkEscalationJobProcessed(ctx context.Context, id uint) error {
	return s.repo.MarkEscalationJobProcessed(ctx, id)
}

func (s *ServiceImpl) CreateNotificationJob(ctx context.Context, job *entities.NotificationJob) error {
	return s.repo.CreateNotificationJob(ctx, job)
}

func (s *ServiceImpl) GetPendingNotificationJobs(ctx context.Context, before time.Time) ([]entities.NotificationJob, error) {
	return s.repo.GetPendingNotificationJobs(ctx, before)
}

func (s *ServiceImpl) MarkNotificationJobProcessed(ctx context.Context, id uint) error {
	return s.repo.MarkNotificationJobProcessed(ctx, id)
}
