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
