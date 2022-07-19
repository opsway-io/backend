package user

import (
	"context"

	"gorm.io/gorm"
)

type Service interface {
	// GetUser returns a user by id.
	GetUser(ctx context.Context, id int64) (*User, error)

	// GetUserByEmail returns a user by email.
	GetUserByEmail(ctx context.Context, email string) (*User, error)

	// CreateUser creates a new user.
	CreateUser(ctx context.Context, user *User) error

	// UpdateUser updates an existing user.
	UpdateUser(ctx context.Context, user *User) error
}

type ServiceImpl struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{db: db}
}

func (s *ServiceImpl) GetUser(ctx context.Context, id int64) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *ServiceImpl) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).Where(User{Email: email}).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *ServiceImpl) CreateUser(ctx context.Context, user *User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *ServiceImpl) UpdateUser(ctx context.Context, user *User) error {
	return s.db.WithContext(ctx).Updates(user).Error
}
