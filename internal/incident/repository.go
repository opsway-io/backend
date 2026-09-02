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
	GetByTeamIDAndStatusPaginated(ctx context.Context, teamID uint, resolved *bool, offset, limit *int) (*[]entities.Incident, error)
	GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]IncidentAndAssertion, error)
	GetActiveByMonitorIDs(ctx context.Context, monitorIDs []uint) ([]entities.Incident, error)
	Upsert(ctx context.Context, incidents *[]entities.Incident) error
	Create(ctx context.Context, incidents *[]entities.Incident) error
	Update(ctx context.Context, incident *entities.Incident) error
	Delete(ctx context.Context, incident *entities.Incident) error
	GetByTeamIDMonitorsIncidentStats(ctx context.Context, teamID uint, start, end string) (*[]entities.MonitorIncident, error)
	CreateOccurrence(ctx context.Context, occurrence *entities.IncidentOccurrence) error
	GetOccurrencesPaginated(ctx context.Context, incidentID uint, offset, limit *int) (*[]entities.IncidentOccurrence, error)
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

func (r *RepositoryImpl) GetByTeamIDAndStatusPaginated(ctx context.Context, teamID uint, resolved *bool, offset, limit *int) (*[]entities.Incident, error) {
	var incidents []entities.Incident
	query := r.db.WithContext(ctx).Where("team_id = ?", teamID)
	if resolved != nil {
		query = query.Where("resolved = ?", *resolved)
	}

	if err := query.Order(
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
	Property *string `gorm:"column:property"`
	Target   *string `gorm:"column:target"`
	Operator *string `gorm:"column:operator"`
}

func (r *RepositoryImpl) GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]IncidentAndAssertion, error) {
	var incidents []IncidentAndAssertion
	if err := r.db.WithContext(
		ctx,
	).Select("incidents.*, ma.property as property, ma.target as target, ma.operator as operator").Where(entities.Incident{
		MonitorID: &monitorID,
	}).Where(
		"resolved = ?", false,
	).Joins(
		"LEFT JOIN monitor_assertions as ma ON ma.id = incidents.monitor_assertion_id",
	).Order(
		"created_at desc",
	// ).Scopes(
	// 	postgres.Paginated(offset, limit),
	).Find(&incidents).Error; err != nil {
		return nil, err
	}

	return &incidents, nil
}

func (r *RepositoryImpl) GetActiveByMonitorIDs(ctx context.Context, monitorIDs []uint) ([]entities.Incident, error) {
	if len(monitorIDs) == 0 {
		return []entities.Incident{}, nil
	}

	var incidents []entities.Incident
	if err := r.db.WithContext(
		ctx,
	).Where("monitor_id IN ? AND resolved = ?", monitorIDs, false).
		Order("created_at desc").
		Find(&incidents).Error; err != nil {
		return nil, err
	}

	return incidents, nil
}

func (r *RepositoryImpl) Upsert(ctx context.Context, incidents *[]entities.Incident) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "monitor_assertion_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at", "resolved", "occurrences", "acknowledged", "acknowledged_by", "acknowledged_at", "root_cause_analysis", "root_cause_notified"}),
	}).Create(incidents).Error
}

func (r *RepositoryImpl) Create(ctx context.Context, incidents *[]entities.Incident) error {
	return r.db.WithContext(ctx).Create(incidents).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, incident *entities.Incident) error {
	result := r.db.WithContext(ctx).Model(incident).Updates(incident)
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

func (r *RepositoryImpl) GetByTeamIDMonitorsIncidentStats(ctx context.Context, teamID uint, start, end string) (*[]entities.MonitorIncident, error) {
	var incidents []entities.MonitorIncident
	err := r.db.WithContext(
		ctx,
	).Table("incidents").Select(`
		monitor_id, 
		count(id) as count`).
		Where("team_id = ?", teamID).
		Where("monitor_id IS NOT NULL").
		Where("created_at BETWEEN ? AND ?", start, end).
		Group("monitor_id").
		Find(&incidents).Error

	if err != nil {
		return nil, err
	}

	return &incidents, nil
}

func (r *RepositoryImpl) CreateOccurrence(ctx context.Context, occurrence *entities.IncidentOccurrence) error {
	return r.db.WithContext(ctx).Create(occurrence).Error
}

func (r *RepositoryImpl) GetOccurrencesPaginated(ctx context.Context, incidentID uint, offset, limit *int) (*[]entities.IncidentOccurrence, error) {
	var occurrences []entities.IncidentOccurrence
	err := r.db.WithContext(ctx).Where("incident_id = ?", incidentID).Order("created_at desc").Scopes(postgres.Paginated(offset, limit)).Find(&occurrences).Error
	if err != nil {
		return nil, err
	}
	return &occurrences, nil
}
