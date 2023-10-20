package changelog

import (
	"context"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Repository interface {
	GetAll(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (changelogs []entities.Changelog, totalCount int, err error)
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
