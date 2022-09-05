package user

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"gorm.io/gorm"
)

var (
	ErrNotFound           = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("user with same email already exists")
)

type Repository interface {
	GetByID(ctx context.Context, id int) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetUsersByTeamID(ctx context.Context, teamID int) (*[]User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (s *RepositoryImpl) GetByID(ctx context.Context, id int) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).Where(User{Email: email}).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) GetUsersByTeamID(ctx context.Context, teamID int) (*[]User, error) {
	var users []User
	if err := s.db.WithContext(ctx).Where(User{TeamID: teamID}).Find(&users).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &users, nil
}

func (s *RepositoryImpl) Create(ctx context.Context, user *User) error {
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.As(err, &postgres.ErrDuplicateEntry) {
			return ErrEmailAlreadyExists
		}

		return err
	}

	return nil
}

func (s *RepositoryImpl) Update(ctx context.Context, user *User) error {
	result := s.db.WithContext(ctx).Updates(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
