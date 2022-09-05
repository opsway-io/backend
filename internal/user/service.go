package user

import (
	"context"
)

type Service interface {
	GetUser(ctx context.Context, id int64) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetUser(ctx context.Context, id int64) (*User, error) {
	return s.repository.GetUser(ctx, id)
}

func (s *ServiceImpl) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.repository.GetUserByEmail(ctx, email)
}

func (s *ServiceImpl) CreateUser(ctx context.Context, user *User) error {
	return s.repository.CreateUser(ctx, user)
}

func (s *ServiceImpl) UpdateUser(ctx context.Context, user *User) error {
	return s.repository.UpdateUser(ctx, user)
}
