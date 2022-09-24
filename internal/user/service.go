package user

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
)

type Service interface {
	GetByID(ctx context.Context, id uint) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
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

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*entities.User, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	return s.repository.GetByEmail(ctx, email)
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
