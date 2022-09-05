package user

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	GetUser(ctx context.Context, id int64) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (s *RepositoryImpl) GetUser(ctx context.Context, id int64) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).Where(User{Email: email}).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) CreateUser(ctx context.Context, user *User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *RepositoryImpl) UpdateUser(ctx context.Context, user *User) error {
	return s.db.WithContext(ctx).Updates(user).Error
}
