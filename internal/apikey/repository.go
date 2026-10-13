package apikey

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, apiKey *entities.APIKey) error
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.APIKey, error)
	GetByHash(ctx context.Context, hash string) (*entities.APIKey, error)
	Delete(ctx context.Context, teamID, keyID uint) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
	}
}

func (r *RepositoryImpl) Create(ctx context.Context, apiKey *entities.APIKey) error {
	return r.db.WithContext(ctx).Create(apiKey).Error
}

func (r *RepositoryImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.APIKey, error) {
	var keys []entities.APIKey
	err := r.db.WithContext(ctx).Where("team_id = ?", teamID).Find(&keys).Error
	if err != nil {
		return nil, err
	}
	return &keys, nil
}

func (r *RepositoryImpl) GetByHash(ctx context.Context, hash string) (*entities.APIKey, error) {
	var key entities.APIKey
	err := r.db.WithContext(ctx).Where("key_hash = ?", hash).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, teamID, keyID uint) error {
	return r.db.WithContext(ctx).Where("team_id = ? AND id = ?", teamID, keyID).Delete(&entities.APIKey{}).Error
}
