package changelog

import (
	"context"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Repository interface {
	GetAll(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (changelogs []entities.Changelog, totalCount int, err error)
	Get(ctx context.Context, teamID, changelogID uint) (entities.Changelog, error)
	Delete(ctx context.Context, teamID, changelogID uint) (err error)
	Create(ctx context.Context, teamID uint, name string) (changelog entities.Changelog, err error)
	Update(ctx context.Context, teamID, changelogID uint, name string) (changelog entities.Changelog, err error)

	GetEntriesWithAuthors(ctx context.Context, teamID, changelogID uint, offset *int, limit *int, query *string) (entries []entities.ChangelogEntry, total_count int, err error)
	// GetEntryWithAuthors(ctx context.Context, teamID, changelogID, entryID uint) (entries entities.ChangelogEntry, err error)
	// DeleteEntry(ctx context.Context, teamID, changelogID, entryID uint) (err error)
	// CreateEntry(ctx context.Context, teamID, changelogID uint, title, content string, authorIDs []uint) (entry entities.ChangelogEntry, err error)
	// UpdateEntry(ctx context.Context, teamID, changelogID, entryID uint, title, content string, authorIDs []uint) (entry entities.ChangelogEntry, err error)
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

func (r *RepositoryImpl) Get(ctx context.Context, teamID, changelogID uint) (entities.Changelog, error) {
	var changelog entities.Changelog

	result := r.db.WithContext(
		ctx,
	).Where(
		"team_id = ? AND id = ?", teamID, changelogID,
	).First(
		&changelog,
	)

	if result.Error != nil {
		return entities.Changelog{}, result.Error
	}

	return changelog, nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, teamID, changelogID uint) error {
	result := r.db.WithContext(
		ctx,
	).Where(
		"team_id = ? AND id = ?", teamID, changelogID,
	).Delete(
		&entities.Changelog{},
	)

	if result.Error != nil {
		return result.Error
	}

	return nil
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

func (r *RepositoryImpl) Update(ctx context.Context, teamID, changelogID uint, name string) (entities.Changelog, error) {
	changelog := entities.Changelog{
		Name: name,
	}

	result := r.db.WithContext(
		ctx,
	).Where(
		"team_id = ? AND id = ?", teamID, changelogID,
	).Updates(
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
