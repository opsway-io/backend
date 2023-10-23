package changelog

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (changelogs []entities.Changelog, totalCount int, err error)
	Get(ctx context.Context, teamID, changelogID uint) (changelogs entities.Changelog, err error)
	Delete(ctx context.Context, teamID, changelogID uint) (err error)
	Create(ctx context.Context, teamID uint, name string) (changelogs entities.Changelog, err error)
	Update(ctx context.Context, teamID, changelogID uint, name string) (changelog entities.Changelog, err error)

	GetEntriesWithAuthors(ctx context.Context, teamID, changelogID uint, offset *int, limit *int, query *string) (entries []entities.ChangelogEntry, total_count int, err error)
	// GetEntryWithAuthors(ctx context.Context, teamID, changelogID, entryID uint) (entries entities.ChangelogEntry, err error)
	// DeleteEntry(ctx context.Context, teamID, changelogID, entryID uint) (err error)
	// CreateEntry(ctx context.Context, teamID, changelogID uint, title, content string, authorIDs []uint) (entry entities.ChangelogEntry, err error)
	// UpdateEntry(ctx context.Context, teamID, changelogID, entryID uint, title, content string, authorIDs []uint) (entry entities.ChangelogEntry, err error)
}

type ServiceImpl struct {
	repo Repository
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{
		repo: NewRepository(db),
	}
}

func (s *ServiceImpl) GetAll(ctx context.Context, teamID uint, offset *int, limit *int, query *string) ([]entities.Changelog, int, error) {
	return s.repo.GetAll(ctx, teamID, offset, limit, query)
}

func (s *ServiceImpl) Get(ctx context.Context, teamID, changelogID uint) (entities.Changelog, error) {
	return s.repo.Get(ctx, teamID, changelogID)
}

func (s *ServiceImpl) Delete(ctx context.Context, teamID, changelogID uint) error {
	return s.repo.Delete(ctx, teamID, changelogID)
}

func (s *ServiceImpl) Create(ctx context.Context, teamID uint, name string) (entities.Changelog, error) {
	return s.repo.Create(ctx, teamID, name)
}

func (s *ServiceImpl) Update(ctx context.Context, teamID, changelogID uint, name string) (entities.Changelog, error) {
	return s.repo.Update(ctx, teamID, changelogID, name)
}

func (s *ServiceImpl) GetEntriesWithAuthors(ctx context.Context, teamID, changelogID uint, offset *int, limit *int, query *string) ([]entities.ChangelogEntry, int, error) {
	return s.repo.GetEntriesWithAuthors(ctx, teamID, changelogID, offset, limit, query)
}
