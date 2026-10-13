package alerting

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("alert rule not found")

type Repository interface {
	GetRulesByTeamID(ctx context.Context, teamID uint) (*[]entities.AlertRule, error)
	GetEnabledRulesByTeamID(ctx context.Context, teamID uint) (*[]entities.AlertRule, error)
	GetRuleByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.AlertRule, error)
	CreateRule(ctx context.Context, rule *entities.AlertRule) error
	UpdateRule(ctx context.Context, rule *entities.AlertRule) error
	DeleteRule(ctx context.Context, teamID, id uint) error

	CreateTrigger(ctx context.Context, trigger *entities.AlertTrigger) error
	GetTriggersByRuleID(ctx context.Context, teamID, ruleID uint, offset, limit *int) (*[]entities.AlertTrigger, int64, error)
	GetTriggersByIncidentID(ctx context.Context, teamID, incidentID uint, offset, limit *int) (*[]entities.AlertTrigger, int64, error)
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetRulesByTeamID(ctx context.Context, teamID uint) (*[]entities.AlertRule, error) {
	var rules []entities.AlertRule
	err := r.db.WithContext(ctx).
		Where(entities.AlertRule{TeamID: teamID}).
		Order("created_at asc").
		Find(&rules).Error

	return &rules, err
}

func (r *RepositoryImpl) GetEnabledRulesByTeamID(ctx context.Context, teamID uint) (*[]entities.AlertRule, error) {
	var rules []entities.AlertRule
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND enabled = ?", teamID, true).
		Order("id asc").
		Find(&rules).Error

	return &rules, err
}

func (r *RepositoryImpl) GetRuleByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.AlertRule, error) {
	var rule entities.AlertRule
	if err := r.db.WithContext(ctx).
		Where(entities.AlertRule{ID: id, TeamID: teamID}).
		First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &rule, nil
}

func (r *RepositoryImpl) CreateRule(ctx context.Context, rule *entities.AlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *RepositoryImpl) UpdateRule(ctx context.Context, rule *entities.AlertRule) error {
	res := r.db.WithContext(ctx).
		Model(&entities.AlertRule{}).
		Where("id = ? AND team_id = ?", rule.ID, rule.TeamID).
		// Select so that zero values (enabled=false) are written too
		Select("name", "condition", "channels", "enabled", "updated_at").
		Updates(rule)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RepositoryImpl) DeleteRule(ctx context.Context, teamID, id uint) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND team_id = ?", id, teamID).
		Delete(&entities.AlertRule{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RepositoryImpl) CreateTrigger(ctx context.Context, trigger *entities.AlertTrigger) error {
	return r.db.WithContext(ctx).Create(trigger).Error
}

func (r *RepositoryImpl) GetTriggersByRuleID(ctx context.Context, teamID, ruleID uint, offset, limit *int) (*[]entities.AlertTrigger, int64, error) {
	return r.getTriggers(ctx, r.db.Where("team_id = ? AND alert_rule_id = ?", teamID, ruleID), offset, limit)
}

func (r *RepositoryImpl) GetTriggersByIncidentID(ctx context.Context, teamID, incidentID uint, offset, limit *int) (*[]entities.AlertTrigger, int64, error) {
	return r.getTriggers(ctx, r.db.Where("team_id = ? AND incident_id = ?", teamID, incidentID), offset, limit)
}

func (r *RepositoryImpl) getTriggers(ctx context.Context, where *gorm.DB, offset, limit *int) (*[]entities.AlertTrigger, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&entities.AlertTrigger{}).
		Where(where).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var triggers []entities.AlertTrigger
	if err := r.db.WithContext(ctx).
		Where(where).
		Order("created_at desc").
		Scopes(postgres.Paginated(offset, limit)).
		Find(&triggers).Error; err != nil {
		return nil, 0, err
	}

	return &triggers, total, nil
}
