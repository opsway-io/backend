package alerting

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

// PolicyRepository stores escalation policies and per-user notification rules.
type PolicyRepository interface {
	GetEscalationPolicy(ctx context.Context, teamID uint) (*entities.EscalationPolicy, error)
	ReplaceEscalationPolicy(ctx context.Context, policy *entities.EscalationPolicy) error
	CountTeamMembers(ctx context.Context, teamID uint, userIDs []uint) (int64, error)

	GetNotificationRules(ctx context.Context, userID uint) (*[]entities.UserNotificationRule, error)
	ReplaceNotificationRules(ctx context.Context, userID uint, rules []entities.UserNotificationRule) error
}

type PolicyRepositoryImpl struct {
	db *gorm.DB
}

func NewPolicyRepository(db *gorm.DB) PolicyRepository {
	return &PolicyRepositoryImpl{db: db}
}

func (r *PolicyRepositoryImpl) GetEscalationPolicy(ctx context.Context, teamID uint) (*entities.EscalationPolicy, error) {
	var policy entities.EscalationPolicy
	if err := r.db.WithContext(ctx).
		Preload("Rotations", func(db *gorm.DB) *gorm.DB { return db.Order("tier asc, id asc") }).
		Where(entities.EscalationPolicy{TeamID: teamID}).
		First(&policy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &policy, nil
}

// ReplaceEscalationPolicy upserts the team's policy and replaces all of its rotations.
func (r *PolicyRepositoryImpl) ReplaceEscalationPolicy(ctx context.Context, policy *entities.EscalationPolicy) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rotations := policy.Rotations
		policy.Rotations = nil

		var existing entities.EscalationPolicy
		err := tx.Where(entities.EscalationPolicy{TeamID: policy.TeamID}).First(&existing).Error

		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(policy).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			policy.ID = existing.ID
			policy.CreatedAt = existing.CreatedAt
			if err := tx.Model(&existing).
				Select("name", "escalation_timeout_minutes", "enforced_channel", "updated_at").
				Updates(policy).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("escalation_policy_id = ?", policy.ID).Delete(&entities.EscalationRotation{}).Error; err != nil {
			return err
		}

		for i := range rotations {
			rotations[i].ID = 0
			rotations[i].EscalationPolicyID = policy.ID
		}

		if len(rotations) > 0 {
			if err := tx.Create(&rotations).Error; err != nil {
				return err
			}
		}

		policy.Rotations = rotations

		return nil
	})
}

func (r *PolicyRepositoryImpl) CountTeamMembers(ctx context.Context, teamID uint, userIDs []uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.TeamUser{}).
		Where("team_id = ? AND user_id IN ?", teamID, userIDs).
		Count(&count).Error

	return count, err
}

func (r *PolicyRepositoryImpl) GetNotificationRules(ctx context.Context, userID uint) (*[]entities.UserNotificationRule, error) {
	var rules []entities.UserNotificationRule
	err := r.db.WithContext(ctx).
		Where(entities.UserNotificationRule{UserID: userID}).
		Order("delay asc, id asc").
		Find(&rules).Error

	return &rules, err
}

func (r *PolicyRepositoryImpl) ReplaceNotificationRules(ctx context.Context, userID uint, rules []entities.UserNotificationRule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&entities.UserNotificationRule{}).Error; err != nil {
			return err
		}

		if len(rules) == 0 {
			return nil
		}

		for i := range rules {
			rules[i].ID = 0
			rules[i].UserID = userID
		}

		return tx.Create(&rules).Error
	})
}
