package team

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("team not found")

type Repository interface {
	GetByID(ctx context.Context, id int) (*Team, error)
	Create(ctx context.Context, team *Team) error
	Update(ctx context.Context, team *Team) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (s *RepositoryImpl) GetByID(ctx context.Context, id int) (*Team, error) {
	var team Team
	if err := s.db.WithContext(ctx).First(&team, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &team, nil
}

func (s *RepositoryImpl) Create(ctx context.Context, team *Team) error {
	return s.db.WithContext(ctx).Create(team).Error
}

func (s *RepositoryImpl) Update(ctx context.Context, team *Team) error {
	result := s.db.WithContext(ctx).Updates(team)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
