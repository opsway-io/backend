package maintenance

import "context"

type Service interface {
	GetByID(ctx context.Context, id uint) (*Maintenance, error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]Maintenance, error)
	Create(ctx context.Context, maintenance *Maintenance) error
	Update(ctx context.Context, maintenance *Maintenance) error
	Delete(ctx context.Context, maintenance *Maintenance) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*Maintenance, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]Maintenance, error) {
	return s.repository.GetByTeamID(ctx, teamID)
}

func (s *ServiceImpl) Create(ctx context.Context, maintenance *Maintenance) error {
	return s.repository.Create(ctx, maintenance)
}

func (s *ServiceImpl) Update(ctx context.Context, maintenance *Maintenance) error {
	return s.repository.Update(ctx, maintenance)
}

func (s *ServiceImpl) Delete(ctx context.Context, maintenance *Maintenance) error {
	return s.repository.Delete(ctx, maintenance)
}
