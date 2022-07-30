package monitor

import (
	"context"

	"gorm.io/gorm"
)

type Service interface {
	Create(ctx context.Context, monitor *Monitor) error
	Update(ctx context.Context, monitor *Monitor) error
	Delete(ctx context.Context, id int) error
	GetByTeamID(ctx context.Context, teamID int, offset int, limit int) ([]Monitor, error)
	GetByTeamIDAndID(ctx context.Context, teamID, id int) (*Monitor, error)
}

type ServiceImpl struct {
	repository Repository
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{
		repository: NewRepository(db),
	}
}

func (s *ServiceImpl) Create(ctx context.Context, m *Monitor) error {
	return s.repository.Create(ctx, m)
}

func (s *ServiceImpl) Update(ctx context.Context, m *Monitor) error {
	return s.repository.Update(ctx, m)
}

func (s *ServiceImpl) Delete(ctx context.Context, id int) error {
	return s.repository.Delete(ctx, id)
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID int, offset int, limit int) ([]Monitor, error) {
	return s.repository.GetByTeamID(ctx, teamID, offset, limit)
}

func (s *ServiceImpl) GetByTeamIDAndID(ctx context.Context, teamID, id int) (*Monitor, error) {
	return s.repository.GetByTeamIDAndID(ctx, teamID, id)
}
