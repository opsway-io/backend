package statuspage

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Repository interface {
	GetByTeamID(ctx context.Context, teamID uint) ([]*entities.StatusPage, error)
	GetByIDAndTeamID(ctx context.Context, id, teamID uint) (*entities.StatusPage, error)
	GetByDomain(ctx context.Context, domain string) (*entities.StatusPage, error)
	CountByTeamID(ctx context.Context, teamID uint) (int64, error)
	Create(ctx context.Context, statusPage *entities.StatusPage) error
	Update(ctx context.Context, statusPage *entities.StatusPage) error
	Delete(ctx context.Context, id, teamID uint) error
	CreateSubscriber(ctx context.Context, sub *entities.StatusPageSubscriber) error
	GetSubscriberByToken(ctx context.Context, token string) (*entities.StatusPageSubscriber, error)
	VerifySubscriber(ctx context.Context, token string) error
	GetVerifiedSubscribers(ctx context.Context, statusPageID uint) ([]entities.StatusPageSubscriber, error)
	ReplaceMonitors(ctx context.Context, statusPage *entities.StatusPage, monitors []entities.Monitor) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
	}
}

func (r *RepositoryImpl) GetByTeamID(ctx context.Context, teamID uint) ([]*entities.StatusPage, error) {
	var statusPages []*entities.StatusPage
	err := r.db.WithContext(ctx).Where("team_id = ?", teamID).Preload("Monitors").Find(&statusPages).Error
	return statusPages, err
}

func (r *RepositoryImpl) GetByIDAndTeamID(ctx context.Context, id, teamID uint) (*entities.StatusPage, error) {
	var statusPage entities.StatusPage
	err := r.db.WithContext(ctx).Where("id = ? AND team_id = ?", id, teamID).Preload("Monitors").First(&statusPage).Error
	return &statusPage, err
}

func (r *RepositoryImpl) GetByDomain(ctx context.Context, domain string) (*entities.StatusPage, error) {
	var statusPage entities.StatusPage
	err := r.db.WithContext(ctx).Where("domain = ?", domain).Preload("Monitors").First(&statusPage).Error
	return &statusPage, err
}

func (r *RepositoryImpl) CountByTeamID(ctx context.Context, teamID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.StatusPage{}).Where("team_id = ?", teamID).Count(&count).Error
	return count, err
}

func (r *RepositoryImpl) Create(ctx context.Context, statusPage *entities.StatusPage) error {
	return r.db.WithContext(ctx).Create(statusPage).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, statusPage *entities.StatusPage) error {
	return r.db.WithContext(ctx).Save(statusPage).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, id, teamID uint) error {
	return r.db.WithContext(ctx).Where("id = ? AND team_id = ?", id, teamID).Delete(&entities.StatusPage{}).Error
}

func (r *RepositoryImpl) CreateSubscriber(ctx context.Context, sub *entities.StatusPageSubscriber) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *RepositoryImpl) GetSubscriberByToken(ctx context.Context, token string) (*entities.StatusPageSubscriber, error) {
	var sub entities.StatusPageSubscriber
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *RepositoryImpl) VerifySubscriber(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Model(&entities.StatusPageSubscriber{}).Where("token = ?", token).Update("verified", true).Error
}

func (r *RepositoryImpl) GetVerifiedSubscribers(ctx context.Context, statusPageID uint) ([]entities.StatusPageSubscriber, error) {
	var subs []entities.StatusPageSubscriber
	err := r.db.WithContext(ctx).Where("status_page_id = ? AND verified = ?", statusPageID, true).Find(&subs).Error
	return subs, err
}

func (r *RepositoryImpl) ReplaceMonitors(ctx context.Context, statusPage *entities.StatusPage, monitors []entities.Monitor) error {
	return r.db.WithContext(ctx).Model(statusPage).Association("Monitors").Replace(monitors)
}
