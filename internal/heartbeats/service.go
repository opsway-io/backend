package heartbeats

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
)

type Service interface {
	GetAllByTeamID(ctx context.Context, teamID uint) ([]entities.Heartbeat, error)
	GetByIDAndTeamID(ctx context.Context, teamID uint, heartbeatID uint) (*entities.Heartbeat, error)
	Create(ctx context.Context, heartbeat *entities.Heartbeat) error
	Update(ctx context.Context, heartbeat *entities.Heartbeat) error
	Delete(ctx context.Context, heartbeat *entities.Heartbeat) error
	Ping(ctx context.Context, heartbeatID uint) error
	GetExpiredHeartbeats(ctx context.Context) ([]entities.Heartbeat, error)
	MarkAsDown(ctx context.Context, heartbeatID uint) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetAllByTeamID(ctx context.Context, teamID uint) ([]entities.Heartbeat, error) {
	return s.repository.GetAllByTeamID(ctx, teamID)
}

func (s *ServiceImpl) GetByIDAndTeamID(ctx context.Context, teamID uint, heartbeatID uint) (*entities.Heartbeat, error) {
	return s.repository.GetByIDAndTeamID(ctx, teamID, heartbeatID)
}

func (s *ServiceImpl) Create(ctx context.Context, heartbeat *entities.Heartbeat) error {
	return s.repository.Create(ctx, heartbeat)
}

func (s *ServiceImpl) Update(ctx context.Context, heartbeat *entities.Heartbeat) error {
	return s.repository.Update(ctx, heartbeat)
}

func (s *ServiceImpl) Delete(ctx context.Context, heartbeat *entities.Heartbeat) error {
	return s.repository.Delete(ctx, heartbeat)
}

func (s *ServiceImpl) Ping(ctx context.Context, heartbeatID uint) error {
	return s.repository.Ping(ctx, heartbeatID)
}

func (s *ServiceImpl) GetExpiredHeartbeats(ctx context.Context) ([]entities.Heartbeat, error) {
	return s.repository.GetExpiredHeartbeats(ctx)
}

func (s *ServiceImpl) MarkAsDown(ctx context.Context, heartbeatID uint) error {
	return s.repository.MarkAsDown(ctx, heartbeatID)
}
