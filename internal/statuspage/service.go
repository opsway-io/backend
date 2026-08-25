package statuspage

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/k8s"
	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("status page not found")
)

type Service interface {
	GetByTeamID(ctx context.Context, teamID uint) ([]*entities.StatusPage, error)
	GetByIDAndTeamID(ctx context.Context, id, teamID uint) (*entities.StatusPage, error)
	GetByDomain(ctx context.Context, domain string) (*entities.StatusPage, error)
	CountByTeamID(ctx context.Context, teamID uint) (int64, error)
	Create(ctx context.Context, statusPage *entities.StatusPage) error
	Update(ctx context.Context, statusPage *entities.StatusPage) error
	Delete(ctx context.Context, id, teamID uint) error
	ReplaceMonitors(ctx context.Context, statusPage *entities.StatusPage, monitors []entities.Monitor) error
	Subscribe(ctx context.Context, statusPageID uint, email string, token string) error
	GetSubscriberByToken(ctx context.Context, token string) (*entities.StatusPageSubscriber, error)
	VerifySubscriber(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	GetVerifiedSubscribers(ctx context.Context, statusPageID uint) ([]entities.StatusPageSubscriber, error)
}

type ServiceImpl struct {
	repository Repository
	k8sService k8s.Service
}

func NewService(repository Repository, k8sService k8s.Service) Service {
	return &ServiceImpl{
		repository: repository,
		k8sService: k8sService,
	}
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID uint) ([]*entities.StatusPage, error) {
	return s.repository.GetByTeamID(ctx, teamID)
}

func (s *ServiceImpl) CountByTeamID(ctx context.Context, teamID uint) (int64, error) {
	return s.repository.CountByTeamID(ctx, teamID)
}

func (s *ServiceImpl) GetByIDAndTeamID(ctx context.Context, id, teamID uint) (*entities.StatusPage, error) {
	statusPage, err := s.repository.GetByIDAndTeamID(ctx, id, teamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return statusPage, nil
}

func (s *ServiceImpl) GetByDomain(ctx context.Context, domain string) (*entities.StatusPage, error) {
	statusPage, err := s.repository.GetByDomain(ctx, domain)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return statusPage, nil
}

func (s *ServiceImpl) Create(ctx context.Context, statusPage *entities.StatusPage) error {
	err := s.repository.Create(ctx, statusPage)
	if err != nil {
		return err
	}
	if statusPage.Domain != "" {
		_ = s.k8sService.UpsertIngress(ctx, statusPage.Domain)
	}
	return nil
}

func (s *ServiceImpl) Update(ctx context.Context, statusPage *entities.StatusPage) error {
	old, err := s.repository.GetByIDAndTeamID(ctx, statusPage.ID, statusPage.TeamID)
	if err == nil && old.Domain != statusPage.Domain && old.Domain != "" {
		_ = s.k8sService.DeleteIngress(ctx, old.Domain)
	}

	err = s.repository.Update(ctx, statusPage)
	if err != nil {
		return err
	}

	if statusPage.Domain != "" {
		_ = s.k8sService.UpsertIngress(ctx, statusPage.Domain)
	}
	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id, teamID uint) error {
	old, err := s.repository.GetByIDAndTeamID(ctx, id, teamID)
	if err == nil && old.Domain != "" {
		_ = s.k8sService.DeleteIngress(ctx, old.Domain)
	}

	err = s.repository.Delete(ctx, id, teamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *ServiceImpl) ReplaceMonitors(ctx context.Context, statusPage *entities.StatusPage, monitors []entities.Monitor) error {
	return s.repository.ReplaceMonitors(ctx, statusPage, monitors)
}

func (s *ServiceImpl) Subscribe(ctx context.Context, statusPageID uint, email string, token string) error {
	sub := &entities.StatusPageSubscriber{
		StatusPageID: statusPageID,
		Email:        email,
		Token:        token,
		Verified:     false, // explicitly set false though it's default
	}
	return s.repository.CreateSubscriber(ctx, sub)
}

func (s *ServiceImpl) GetSubscriberByToken(ctx context.Context, token string) (*entities.StatusPageSubscriber, error) {
	return s.repository.GetSubscriberByToken(ctx, token)
}

func (s *ServiceImpl) VerifySubscriber(ctx context.Context, token string) error {
	return s.repository.VerifySubscriber(ctx, token)
}

func (s *ServiceImpl) Unsubscribe(ctx context.Context, token string) error {
	return s.repository.Unsubscribe(ctx, token)
}

func (s *ServiceImpl) GetVerifiedSubscribers(ctx context.Context, statusPageID uint) ([]entities.StatusPageSubscriber, error) {
	return s.repository.GetVerifiedSubscribers(ctx, statusPageID)
}
