package incident

import "context"

type Service interface {
	GetByID(ctx context.Context, id uint) (*Incident, error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]Incident, error)
	Create(ctx context.Context, incident *Incident) error
	Update(ctx context.Context, incident *Incident) error
	Delete(ctx context.Context, incident *Incident) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*Incident, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]Incident, error) {
	return s.repository.GetByTeamID(ctx, teamID)
}

func (s *ServiceImpl) Create(ctx context.Context, incident *Incident) error {
	return s.repository.Create(ctx, incident)
}

func (s *ServiceImpl) Update(ctx context.Context, incident *Incident) error {
	return s.repository.Update(ctx, incident)
}

func (s *ServiceImpl) Delete(ctx context.Context, incident *Incident) error {
	return s.repository.Delete(ctx, incident)
}
