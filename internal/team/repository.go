package team

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	GetTeamByID(ctx context.Context, id int) (*Team, error)
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (s *RepositoryImpl) GetTeamByID(ctx context.Context, id int) (*Team, error) {
	var team Team
	if err := s.db.WithContext(ctx).First(&team, id).Error; err != nil {
		return nil, err
	}

	return &team, nil
}
