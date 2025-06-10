package incident

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("incident not found")

type Repository interface {
	GetByID(ctx context.Context, id uint) (*entities.Incident, error)
	GetByTeamIDPaginated(ctx context.Context, teamID uint, offset, limit *int) (*[]entities.Incident, error)
	GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]IncidentAndAssertion, error)
	Upsert(ctx context.Context, incidents *[]entities.Incident) error
	Create(ctx context.Context, incidents *[]entities.Incident) error
	Update(ctx context.Context, incident *entities.Incident) error
	Delete(ctx context.Context, incident *entities.Incident) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetByID(ctx context.Context, id uint) (*entities.Incident, error) {
	var incident entities.Incident
	if err := r.db.WithContext(
		ctx,
	).Where(entities.Incident{
		ID: id,
	}).First(&incident).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &incident, nil
}

func (r *RepositoryImpl) GetByTeamIDPaginated(ctx context.Context, teamID uint, offset, limit *int) (*[]entities.Incident, error) {
	var incidents []entities.Incident
	if err := r.db.WithContext(
		ctx,
	).Where(entities.Incident{
		TeamID: teamID,
	}).Order(
		"created_at desc",
	).Scopes(
		postgres.Paginated(offset, limit),
	).Find(&incidents).Error; err != nil {
		return nil, err
	}

	return &incidents, nil
}

type IncidentAndAssertion struct {
	entities.Incident
	Property string `gorm:"column:property"`
	Target   string `gorm:"column:target"`
}

func (r *RepositoryImpl) GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]IncidentAndAssertion, error) {
	var incidents []IncidentAndAssertion
	if err := r.db.WithContext(
		ctx,
	).Select("incidents.*, ma.property as property, ma.target as target").Where(entities.Incident{
		MonitorID: monitorID,
	}).Where(
		"resolved = ?", false,
	).Joins(
		"INNER JOIN monitor_assertions as ma ON ma.id = incidents.monitor_assertion_id",
	).Order(
		"created_at desc",
	// ).Scopes(
	// 	postgres.Paginated(offset, limit),
	).Find(&incidents).Error; err != nil {
		return nil, err
	}

	return &incidents, nil
}

func (r *RepositoryImpl) Upsert(ctx context.Context, incidents *[]entities.Incident) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "monitor_assertion_id"}, {Name: "resolved"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).Create(incidents).Error
}

func (r *RepositoryImpl) Create(ctx context.Context, incidents *[]entities.Incident) error {
	return r.db.WithContext(ctx).Create(incidents).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, incident *entities.Incident) error {
	result := r.db.WithContext(ctx).Updates(incident)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, incident *entities.Incident) error {
	result := r.db.WithContext(ctx).Delete(incident)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
