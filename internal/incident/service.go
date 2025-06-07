package incident

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
)

type Service interface {
	GetByID(ctx context.Context, id uint) (*entities.Incident, error)
	GetByTeamIDPaginated(ctx context.Context, teamID uint, offset, limit *int) (*[]entities.Incident, error)
	Upsert(ctx context.Context, incidents *[]entities.Incident) error
	Create(ctx context.Context, incidents *[]entities.Incident) error
	Update(ctx context.Context, incident *entities.Incident) error
	Delete(ctx context.Context, incident *entities.Incident) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*entities.Incident, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) GetByTeamIDPaginated(ctx context.Context, teamID uint, offset, limit *int) (*[]entities.Incident, error) {
	return s.repository.GetByTeamIDPaginated(ctx, teamID, offset, limit)
}

func (s *ServiceImpl) Upsert(ctx context.Context, incidents *[]entities.Incident) error {
	return s.repository.Upsert(ctx, incidents)
}

func (s *ServiceImpl) Create(ctx context.Context, incidents *[]entities.Incident) error {
	return s.repository.Create(ctx, incidents)
}

func (s *ServiceImpl) Update(ctx context.Context, incident *entities.Incident) error {
	return s.repository.Update(ctx, incident)
}

func (s *ServiceImpl) Delete(ctx context.Context, incident *entities.Incident) error {
	return s.repository.Delete(ctx, incident)
}
