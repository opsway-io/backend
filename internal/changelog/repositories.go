package changelog

import (
	"context"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Repository interface {
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

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
	}
}

type ChangelogWithTotalCount struct {
	entities.Changelog
	TotalCount int64 `gorm:"column:total_count"`
}

func (r *RepositoryImpl) GetAll(ctx context.Context, teamID uint, offset *int, limit *int, query *string) ([]entities.Changelog, int, error) {
	var changelogs []entities.Changelog
	var totalCount int64

	result := r.db.WithContext(
		ctx,
	).Scopes(
		postgres.Paginated(offset, limit),
		postgres.IncludeTotalCount("total_count"),
		postgres.Search([]string{"name"}, query),
	).Where(
		"team_id = ?", teamID,
	).Find(
		&changelogs,
	)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return changelogs, int(totalCount), nil
}

func (r *RepositoryImpl) Create(ctx context.Context, teamID uint, name string) (entities.Changelog, error) {
	changelog := entities.Changelog{
		TeamID: teamID,
		Name:   name,
	}

	result := r.db.WithContext(
		ctx,
	).Create(
		&changelog,
	)

	if result.Error != nil {
		return entities.Changelog{}, result.Error
	}

	return changelog, nil
}

func (r *RepositoryImpl) GetEntriesWithAuthors(ctx context.Context, teamID, changelogID uint, offset *int, limit *int, query *string) ([]entities.ChangelogEntry, int, error) {
	var entries []entities.ChangelogEntry
	var totalCount int64

	result := r.db.WithContext(
		ctx,
	).Scopes(
		postgres.Paginated(offset, limit),
		postgres.IncludeTotalCount("total_count"),
		postgres.Search([]string{"title", "content"}, query),
	).Where(
		"team_id = ? AND changelog_id = ?", teamID, changelogID,
	).Find(
		&entries,
	)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return entries, int(totalCount), nil
}
