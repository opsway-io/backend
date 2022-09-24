package team

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var (
	ErrNotFound          = errors.New("team not found")
	ErrUserNotFound      = errors.New("team user not found")
	ErrNameAlreadyExists = errors.New("team name already exists")
)

type Repository interface {
	GetByID(ctx context.Context, id uint) (*entities.Team, error)
	GetUsersByID(ctx context.Context, id uint) (*[]entities.User, error)
	GetUserRole(ctx context.Context, teamID, userID uint) (*entities.Role, error)
	Create(ctx context.Context, team *entities.Team) error
	Update(ctx context.Context, team *entities.Team) error
	Delete(ctx context.Context, id uint) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (s *RepositoryImpl) GetByID(ctx context.Context, id uint) (*entities.Team, error) {
	var team entities.Team
	if err := s.db.WithContext(ctx).First(&team, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &team, nil
}

func (s *RepositoryImpl) GetUsersByID(ctx context.Context, id uint) (*[]entities.User, error) {
	var users []entities.User
	if err := s.db.WithContext(
		ctx,
	).Model(
		&entities.Team{
			ID: id,
		},
	).Association(
		"Users",
	).Find(
		&users,
	); err != nil {
		return nil, err
	}

	return &users, nil
}

func (s *RepositoryImpl) Create(ctx context.Context, team *entities.Team) error {
	if err := s.db.WithContext(ctx).Create(team).Error; err != nil {
		if errors.As(err, &postgres.ErrDuplicateEntry) {
			return ErrNameAlreadyExists
		}

		return err
	}

	return nil
}

func (s *RepositoryImpl) Update(ctx context.Context, team *entities.Team) error {
	result := s.db.WithContext(ctx).Updates(team)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *RepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&entities.Team{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *RepositoryImpl) GetUserRole(ctx context.Context, teamID, userID uint) (*entities.Role, error) {
	var userRole entities.UserRole
	if err := s.db.WithContext(ctx).Where(
		entities.UserRole{
			TeamID: teamID,
			UserID: userID,
		},
	).First(&userRole).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &userRole.Role, nil
}
