package alerting

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Repository interface {
	GetAllByTeamID(ctx context.Context, teamID uint) ([]entities.AlertRule, error)
	GetByIDAndTeamID(ctx context.Context, teamID uint, ruleID uint) (*entities.AlertRule, error)
	Create(ctx context.Context, rule *entities.AlertRule) error
	Update(ctx context.Context, rule *entities.AlertRule) error
	Delete(ctx context.Context, rule *entities.AlertRule) error

	// Triggers
	GetTriggersByRuleID(ctx context.Context, teamID uint, ruleID uint, offset *int, limit *int) (int, []entities.AlertTrigger, error)
	GetTriggersByIncidentID(ctx context.Context, teamID uint, incidentID uint, offset *int, limit *int) (int, []entities.AlertTrigger, error)
	CreateTrigger(ctx context.Context, trigger *entities.AlertTrigger) error
	HasTriggered(ctx context.Context, incidentID uint, ruleID uint) (bool, error)
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
	}
}

func (r *RepositoryImpl) GetAllByTeamID(ctx context.Context, teamID uint) ([]entities.AlertRule, error) {
	var rules []entities.AlertRule
	err := r.db.WithContext(ctx).Where("team_id = ?", teamID).Find(&rules).Error
	return rules, err
}

func (r *RepositoryImpl) GetByIDAndTeamID(ctx context.Context, teamID uint, ruleID uint) (*entities.AlertRule, error) {
	var rule entities.AlertRule
	err := r.db.WithContext(ctx).Where("team_id = ? AND id = ?", teamID, ruleID).First(&rule).Error
	return &rule, err
}

func (r *RepositoryImpl) Create(ctx context.Context, rule *entities.AlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, rule *entities.AlertRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, rule *entities.AlertRule) error {
	return r.db.WithContext(ctx).Delete(rule).Error
}

func (r *RepositoryImpl) GetTriggersByRuleID(ctx context.Context, teamID uint, ruleID uint, offset *int, limit *int) (int, []entities.AlertTrigger, error) {
	var triggers []entities.AlertTrigger
	var totalCount int64

	baseQuery := r.db.WithContext(ctx).Model(&entities.AlertTrigger{}).Where("team_id = ? AND alert_rule_id = ?", teamID, ruleID)

	err := baseQuery.Count(&totalCount).Error
	if err != nil {
		return 0, nil, err
	}

	query := baseQuery.Order("created_at DESC")
	if offset != nil {
		query = query.Offset(*offset)
	}
	if limit != nil {
		query = query.Limit(*limit)
	}
	err = query.Find(&triggers).Error
	return int(totalCount), triggers, err
}

func (r *RepositoryImpl) GetTriggersByIncidentID(ctx context.Context, teamID uint, incidentID uint, offset *int, limit *int) (int, []entities.AlertTrigger, error) {
	var triggers []entities.AlertTrigger
	var totalCount int64

	baseQuery := r.db.WithContext(ctx).Model(&entities.AlertTrigger{}).Where("team_id = ? AND incident_id = ?", teamID, incidentID)

	err := baseQuery.Count(&totalCount).Error
	if err != nil {
		return 0, nil, err
	}

	query := baseQuery.Order("created_at DESC")
	if offset != nil {
		query = query.Offset(*offset)
	}
	if limit != nil {
		query = query.Limit(*limit)
	}
	err = query.Find(&triggers).Error
	return int(totalCount), triggers, err
}

func (r *RepositoryImpl) CreateTrigger(ctx context.Context, trigger *entities.AlertTrigger) error {
	return r.db.WithContext(ctx).Create(trigger).Error
}

func (r *RepositoryImpl) HasTriggered(ctx context.Context, incidentID uint, ruleID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.AlertTrigger{}).
		Where("incident_id = ? AND alert_rule_id = ?", incidentID, ruleID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
