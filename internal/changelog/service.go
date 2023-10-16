package changelog

import "gorm.io/gorm"

type Service interface {
	// GetAll(ctx context.Context, teamID uint, offset, limit int) ([]entities.Changelog, int, error)
	// Get(ctx context.Context, teamID, changelogID uint) (entities.Changelog, error)
	// Delete(ctx context.Context, teamID, changelogID uint) error
	// Create(ctx context.Context, teamID uint, name string) (entities.Changelog, error)
	// Update(ctx context.Context, teamID, changelogID uint, name string) (entities.Changelog, error)

	// GetEntriesWithAuthors(ctx context.Context, teamID, changelogID uint, offset, limit int) ([]entities.ChangelogEntry, int, error)
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
