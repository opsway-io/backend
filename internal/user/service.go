package user

import (
	"context"
)

type Service interface {
	GetByID(ctx context.Context, id int) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByTeamID(ctx context.Context, teamID int) (*[]User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, id int) (*User, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.repository.GetByEmail(ctx, email)
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID int) (*[]User, error) {
	return s.repository.GetUsersByTeamID(ctx, teamID)
}

func (s *ServiceImpl) Create(ctx context.Context, user *User) error {
	return s.repository.Create(ctx, user)
}

func (s *ServiceImpl) Update(ctx context.Context, user *User) error {
	return s.repository.Update(ctx, user)
}
