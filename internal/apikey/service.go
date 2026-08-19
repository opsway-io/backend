package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/opsway-io/backend/internal/entities"
)

type Service interface {
	Create(ctx context.Context, teamID uint, name string) (plaintextKey string, err error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.APIKey, error)
	GetByPlaintext(ctx context.Context, plaintextKey string) (*entities.APIKey, error)
	Delete(ctx context.Context, teamID, keyID uint) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) Create(ctx context.Context, teamID uint, name string) (string, error) {
	// Generate random 32 byte key
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	plaintextKey := hex.EncodeToString(b)
	
	// Hash the key for storage
	hash := sha256.Sum256([]byte(plaintextKey))
	keyHash := hex.EncodeToString(hash[:])

	apiKey := &entities.APIKey{
		TeamID:  teamID,
		Name:    name,
		KeyHash: keyHash,
	}

	err = s.repository.Create(ctx, apiKey)
	if err != nil {
		return "", err
	}

	return plaintextKey, nil
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.APIKey, error) {
	return s.repository.GetByTeamID(ctx, teamID)
}

func (s *ServiceImpl) GetByPlaintext(ctx context.Context, plaintextKey string) (*entities.APIKey, error) {
	hash := sha256.Sum256([]byte(plaintextKey))
	keyHash := hex.EncodeToString(hash[:])
	
	return s.repository.GetByHash(ctx, keyHash)
}

func (s *ServiceImpl) Delete(ctx context.Context, teamID, keyID uint) error {
	return s.repository.Delete(ctx, teamID, keyID)
}
