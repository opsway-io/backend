package escalation

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Repository interface {
	GetPolicyByTeamID(ctx context.Context, teamID uint) (*entities.EscalationPolicy, error)
	CreatePolicy(ctx context.Context, policy *entities.EscalationPolicy) error
	UpdatePolicy(ctx context.Context, policy *entities.EscalationPolicy) error
	
	GetRotationsByPolicyID(ctx context.Context, policyID uint) ([]entities.OnCallRotation, error)
	GetRotationsByPolicyIDAndTier(ctx context.Context, policyID uint, tier int) ([]entities.OnCallRotation, error)
	SetRotations(ctx context.Context, policyID uint, rotations []entities.OnCallRotation) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetPolicyByTeamID(ctx context.Context, teamID uint) (*entities.EscalationPolicy, error) {
	var policy entities.EscalationPolicy
	err := r.db.WithContext(ctx).Where("team_id = ?", teamID).First(&policy).Error
	return &policy, err
}

func (r *RepositoryImpl) CreatePolicy(ctx context.Context, policy *entities.EscalationPolicy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}

func (r *RepositoryImpl) UpdatePolicy(ctx context.Context, policy *entities.EscalationPolicy) error {
	return r.db.WithContext(ctx).Save(policy).Error
}

func (r *RepositoryImpl) GetRotationsByPolicyID(ctx context.Context, policyID uint) ([]entities.OnCallRotation, error) {
	var rotations []entities.OnCallRotation
	err := r.db.WithContext(ctx).Where("escalation_policy_id = ?", policyID).Order("tier asc").Find(&rotations).Error
	return rotations, err
}

func (r *RepositoryImpl) GetRotationsByPolicyIDAndTier(ctx context.Context, policyID uint, tier int) ([]entities.OnCallRotation, error) {
	var rotations []entities.OnCallRotation
	err := r.db.WithContext(ctx).Where("escalation_policy_id = ? AND tier = ?", policyID, tier).Find(&rotations).Error
	return rotations, err
}

func (r *RepositoryImpl) SetRotations(ctx context.Context, policyID uint, rotations []entities.OnCallRotation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("escalation_policy_id = ?", policyID).Delete(&entities.OnCallRotation{}).Error; err != nil {
			return err
		}
		if len(rotations) > 0 {
			if err := tx.Create(&rotations).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
