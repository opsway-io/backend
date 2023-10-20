package changelog

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (changelogs []entities.Changelog, totalCount int, err error)
	// Get(ctx context.Context, teamID, changelogID uint) (entities.Changelog, error)
	// Delete(ctx context.Context, teamID, changelogID uint) error
	Create(ctx context.Context, teamID uint, name string) (entities.Changelog, error)
	// Update(ctx context.Context, teamID, changelogID uint, name string) (entities.Changelog, error)

	GetEntriesWithAuthors(ctx context.Context, teamID, changelogID uint, offset *int, limit *int, query *string) (entries []entities.ChangelogEntry, total_count int, err error)
	// GetEntryWithAuthors(ctx context.Context, teamID, changelogID, entryID uint) (entities.ChangelogEntry, error)
	// DeleteEntry(ctx context.Context, teamID, changelogID, entryID uint) error
	// CreateEntry(ctx context.Context, teamID, changelogID uint, title, content string, authorIDs []uint) (entities.ChangelogEntry, error)
	// UpdateEntry(ctx context.Context, teamID, changelogID, entryID uint, title, content string, authorIDs []uint) (entities.ChangelogEntry, error)
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

func (s *ServiceImpl) Create(ctx context.Context, teamID uint, name string) (entities.Changelog, error) {
	return s.repo.Create(ctx, teamID, name)
}

func (s *ServiceImpl) GetEntriesWithAuthors(ctx context.Context, teamID, changelogID uint, offset *int, limit *int, query *string) ([]entities.ChangelogEntry, int, error) {
	return s.repo.GetEntriesWithAuthors(ctx, teamID, changelogID, offset, limit, query)
}
