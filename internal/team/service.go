package team

import (
	"context"
)

type Service interface {
	GetTeamByID(ctx context.Context, id int) (*Team, error)
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetTeamByID(ctx context.Context, id int) (*Team, error) {
	return s.repository.GetTeamByID(ctx, id)
}
