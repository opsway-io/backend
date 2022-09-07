package team

import (
	"context"
)

type Service interface {
	GetByID(ctx context.Context, id uint) (*Team, error)
	Create(ctx context.Context, team *Team) error
	Update(ctx context.Context, team *Team) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*Team, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) Create(ctx context.Context, team *Team) error {
	return s.repository.Create(ctx, team)
}

func (s *ServiceImpl) Update(ctx context.Context, team *Team) error {
	return s.repository.Update(ctx, team)
}
