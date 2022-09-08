package user

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var (
	ErrNotFound           = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("user with same email already exists")
)

type Repository interface {
	GetByID(ctx context.Context, id uint) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetUsersByTeamID(ctx context.Context, teamID uint) (*[]entities.User, error)
	Create(ctx context.Context, user *entities.User) error
	Update(ctx context.Context, user *entities.User) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (s *RepositoryImpl) GetByID(ctx context.Context, id uint) (*entities.User, error) {
	var user entities.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	if err := s.db.WithContext(ctx).Where(entities.User{Email: email}).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) GetUsersByTeamID(ctx context.Context, teamID uint) (*[]entities.User, error) {
	// TODO

	return nil, nil
}

func (s *RepositoryImpl) Create(ctx context.Context, user *entities.User) error {
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.As(err, &postgres.ErrDuplicateEntry) {
			return ErrEmailAlreadyExists
		}

		return err
	}

	return nil
}

func (s *RepositoryImpl) Update(ctx context.Context, user *entities.User) error {
	result := s.db.WithContext(ctx).Updates(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
