package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("api key not found")

type Service interface {
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.ApiKey, error)
	// Create generates a new key. The plaintext key is returned once and never stored.
	Create(ctx context.Context, teamID, userID uint, name string) (key *entities.ApiKey, plaintext string, err error)
	Delete(ctx context.Context, teamID, id uint) error
	// Verify looks up the key matching a plaintext key and records its usage.
	Verify(ctx context.Context, plaintext string) (*entities.ApiKey, error)
}

type ServiceImpl struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{db: db}
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.ApiKey, error) {
	var keys []entities.ApiKey
	err := s.db.WithContext(ctx).
		Where(entities.ApiKey{TeamID: teamID}).
		Order("created_at desc").
		Find(&keys).Error

	return &keys, err
}

func (s *ServiceImpl) Create(ctx context.Context, teamID, userID uint, name string) (*entities.ApiKey, string, error) {
	plaintext, err := GenerateKey()
	if err != nil {
		return nil, "", err
	}

	key := &entities.ApiKey{
		TeamID:          teamID,
		CreatedByUserID: userID,
		Name:            name,
		Prefix:          DisplayPrefix(plaintext),
		Hash:            HashKey(plaintext),
	}

	if err := s.db.WithContext(ctx).Create(key).Error; err != nil {
		return nil, "", err
	}

	return key, plaintext, nil
}

func (s *ServiceImpl) Delete(ctx context.Context, teamID, id uint) error {
	res := s.db.WithContext(ctx).
		Where("id = ? AND team_id = ?", id, teamID).
		Delete(&entities.ApiKey{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *ServiceImpl) Verify(ctx context.Context, plaintext string) (*entities.ApiKey, error) {
	if !LooksLikeKey(plaintext) {
		return nil, ErrNotFound
	}

	var key entities.ApiKey
	if err := s.db.WithContext(ctx).
		Where(entities.ApiKey{Hash: HashKey(plaintext)}).
		First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	now := time.Now()
	_ = s.db.WithContext(ctx).Model(&key).UpdateColumn("last_used_at", now).Error
	key.LastUsedAt = &now

	return &key, nil
}
