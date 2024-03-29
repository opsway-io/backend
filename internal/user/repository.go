package user

import (
	"context"
	"errors"
	"strings"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound           = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("user with same email already exists")
)

type Repository interface {
	GetUserByID(ctx context.Context, userID uint) (*entities.User, error)
	GetUserAndTeamsByUserID(ctx context.Context, userID uint) (*entities.User, error)
	GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error)
	Create(ctx context.Context, user *entities.User) error
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uint) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetUserByID(ctx context.Context, userID uint) (*entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).First(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) GetUserAndTeamsByUserID(ctx context.Context, userID uint) (*entities.User, error) {
	var user entities.User
	if err := s.db.WithContext(ctx).Preload("Teams").Where(entities.User{ID: userID}).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	if err := s.db.WithContext(ctx).Preload("Teams").Where(entities.User{Email: strings.ToLower(email)}).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &user, nil
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
	result := s.db.WithContext(ctx).Model(user).Updates(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *RepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Select(clause.Associations).Delete(&entities.User{
		ID: id,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
