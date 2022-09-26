package user

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
)

type Service interface {
	GetUserAndTeamsByUserID(ctx context.Context, userId uint) (*entities.User, error)
	GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error)
	Create(ctx context.Context, user *entities.User) error
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uint) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetUserAndTeamsByUserID(ctx context.Context, userId uint) (*entities.User, error) {
	return s.repository.GetUserAndTeamsByUserID(ctx, userId)
}

func (s *ServiceImpl) GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error) {
	return s.repository.GetUserAndTeamsByEmailAddress(ctx, email)
}

func (s *ServiceImpl) Create(ctx context.Context, user *entities.User) error {
	return s.repository.Create(ctx, user)
}

func (s *ServiceImpl) Update(ctx context.Context, user *entities.User) error {
	return s.repository.Update(ctx, user)
}

func (s *ServiceImpl) Delete(ctx context.Context, id uint) error {
	return s.repository.Delete(ctx, id)
}
