package alerting

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
)

type Service interface {
	GetAllByTeamID(ctx context.Context, teamID uint) ([]entities.AlertRule, error)
	GetByIDAndTeamID(ctx context.Context, teamID uint, ruleID uint) (*entities.AlertRule, error)
	Create(ctx context.Context, rule *entities.AlertRule) error
	Update(ctx context.Context, rule *entities.AlertRule) error
	Delete(ctx context.Context, rule *entities.AlertRule) error

	// Triggers
	GetTriggersByRuleID(ctx context.Context, teamID uint, ruleID uint, offset *int, limit *int) ([]entities.AlertTrigger, error)
	GetTriggersByIncidentID(ctx context.Context, teamID uint, incidentID uint, offset *int, limit *int) ([]entities.AlertTrigger, error)
	CreateTrigger(ctx context.Context, trigger *entities.AlertTrigger) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetAllByTeamID(ctx context.Context, teamID uint) ([]entities.AlertRule, error) {
	return s.repository.GetAllByTeamID(ctx, teamID)
}

func (s *ServiceImpl) GetByIDAndTeamID(ctx context.Context, teamID uint, ruleID uint) (*entities.AlertRule, error) {
	return s.repository.GetByIDAndTeamID(ctx, teamID, ruleID)
}

func (s *ServiceImpl) Create(ctx context.Context, rule *entities.AlertRule) error {
	return s.repository.Create(ctx, rule)
}

func (s *ServiceImpl) Update(ctx context.Context, rule *entities.AlertRule) error {
	return s.repository.Update(ctx, rule)
}

func (s *ServiceImpl) Delete(ctx context.Context, rule *entities.AlertRule) error {
	return s.repository.Delete(ctx, rule)
}

func (s *ServiceImpl) GetTriggersByRuleID(ctx context.Context, teamID uint, ruleID uint, offset *int, limit *int) ([]entities.AlertTrigger, error) {
	return s.repository.GetTriggersByRuleID(ctx, teamID, ruleID, offset, limit)
}

func (s *ServiceImpl) GetTriggersByIncidentID(ctx context.Context, teamID uint, incidentID uint, offset *int, limit *int) ([]entities.AlertTrigger, error) {
	return s.repository.GetTriggersByIncidentID(ctx, teamID, incidentID, offset, limit)
}

func (s *ServiceImpl) CreateTrigger(ctx context.Context, trigger *entities.AlertTrigger) error {
	return s.repository.CreateTrigger(ctx, trigger)
}
