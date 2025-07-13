package report

import (
	"context"

	"gorm.io/gorm"
)

type Service interface {
	GetResportsByTeam(ctx context.Context, teamID uint) (string, error)
}

type ServiceImpl struct {
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{}
}

func (s *ServiceImpl) GetResportsByTeam(ctx context.Context, teamID uint) (string, error) {
	return "", nil
}
