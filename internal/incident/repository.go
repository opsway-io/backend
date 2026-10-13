package incident

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgconn"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound                   = errors.New("incident not found")
	ErrUnresolvedIncidentExists   = errors.New("an unresolved incident already exists for this assertion")
	legacyUnresolvedIncidentIndex = "unresolved_incident"
)

type Repository interface {
	GetByID(ctx context.Context, id uint) (*entities.Incident, error)
	GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.Incident, error)
	GetByTeamIDPaginated(ctx context.Context, teamID uint, resolved *bool, offset, limit *int) (*[]IncidentWithOccurrences, error)
	GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]IncidentAndAssertion, error)
	GetOccurrencesPaginated(ctx context.Context, incidentID uint, offset, limit *int) (*[]entities.IncidentOccurrence, int64, error)
	CountOccurrences(ctx context.Context, incidentID uint) (int64, error)

	// CreateOrTouch creates the incident if no unresolved incident exists for its assertion,
	// otherwise it bumps the existing one. In both cases an occurrence is recorded.
	// The incident is updated in place with the stored values; created reports whether it is new.
	CreateOrTouch(ctx context.Context, incident *entities.Incident) (created bool, err error)
	Create(ctx context.Context, incidents *[]entities.Incident) error
	UpdateFields(ctx context.Context, teamID, id uint, fields map[string]any) error
	Delete(ctx context.Context, incident *entities.Incident) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

// Migrate drops the legacy unique index on (monitor_assertion_id, resolved), which only allowed a
// single resolved incident per assertion. It is replaced by a partial unique index on unresolved incidents.
func Migrate(db *gorm.DB) error {
	m := db.Migrator()
	if m.HasIndex(&entities.Incident{}, legacyUnresolvedIncidentIndex) {
		return m.DropIndex(&entities.Incident{}, legacyUnresolvedIncidentIndex)
	}

	return nil
}

func (r *RepositoryImpl) GetByID(ctx context.Context, id uint) (*entities.Incident, error) {
	return r.get(ctx, entities.Incident{ID: id})
}

func (r *RepositoryImpl) GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.Incident, error) {
	return r.get(ctx, entities.Incident{ID: id, TeamID: teamID})
}

func (r *RepositoryImpl) get(ctx context.Context, where entities.Incident) (*entities.Incident, error) {
	var incident entities.Incident
	if err := r.db.WithContext(ctx).
		Preload("AcknowledgedBy").
		Preload("ResolvedBy").
		Where(where).
		First(&incident).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &incident, nil
}

type IncidentWithOccurrences struct {
	entities.Incident
	OccurrenceCount int64 `gorm:"column:occurrence_count"`
}

func (r *RepositoryImpl) GetByTeamIDPaginated(ctx context.Context, teamID uint, resolved *bool, offset, limit *int) (*[]IncidentWithOccurrences, error) {
	var incidents []IncidentWithOccurrences

	q := r.db.WithContext(ctx).
		Model(&entities.Incident{}).
		Select("incidents.*, (SELECT count(*) FROM incident_occurrences io WHERE io.incident_id = incidents.id) AS occurrence_count").
		Preload("AcknowledgedBy").
		Preload("ResolvedBy").
		Where("incidents.team_id = ?", teamID)

	if resolved != nil {
		q = q.Where("incidents.resolved = ?", *resolved)
	}

	if err := q.Order("incidents.created_at desc").
		Scopes(postgres.Paginated(offset, limit)).
		Find(&incidents).Error; err != nil {
		return nil, err
	}

	return &incidents, nil
}

type IncidentAndAssertion struct {
	entities.Incident
	Property string `gorm:"column:property"`
	Target   string `gorm:"column:target"`
	Operator string `gorm:"column:operator"`
}

func (r *RepositoryImpl) GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]IncidentAndAssertion, error) {
	var incidents []IncidentAndAssertion
	if err := r.db.WithContext(ctx).
		Model(&entities.Incident{}).
		Select("incidents.*, ma.property as property, ma.target as target, ma.operator as operator").
		Preload("AcknowledgedBy").
		Preload("ResolvedBy").
		Where("incidents.monitor_id = ?", monitorID).
		Where("incidents.resolved = ?", false).
		Joins("INNER JOIN monitor_assertions as ma ON ma.id = incidents.monitor_assertion_id").
		Order("incidents.created_at desc").
		Scopes(postgres.Paginated(offset, limit)).
		Find(&incidents).Error; err != nil {
		return nil, err
	}

	return &incidents, nil
}

func (r *RepositoryImpl) GetOccurrencesPaginated(ctx context.Context, incidentID uint, offset, limit *int) (*[]entities.IncidentOccurrence, int64, error) {
	total, err := r.CountOccurrences(ctx, incidentID)
	if err != nil {
		return nil, 0, err
	}

	var occurrences []entities.IncidentOccurrence
	if err := r.db.WithContext(ctx).
		Where(entities.IncidentOccurrence{IncidentID: incidentID}).
		Order("created_at desc").
		Scopes(postgres.Paginated(offset, limit)).
		Find(&occurrences).Error; err != nil {
		return nil, 0, err
	}

	return &occurrences, total, nil
}

func (r *RepositoryImpl) CountOccurrences(ctx context.Context, incidentID uint) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&entities.IncidentOccurrence{}).
		Where("incident_id = ?", incidentID).
		Count(&total).Error

	return total, err
}

func (r *RepositoryImpl) CreateOrTouch(ctx context.Context, incident *entities.Incident) (bool, error) {
	created := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{
			Columns:     []clause.Column{{Name: "monitor_assertion_id"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Eq{Column: clause.Column{Name: "resolved"}, Value: false}}},
			DoNothing:   true,
		}).Create(incident)
		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected == 0 {
			// An unresolved incident already exists, touch it
			var existing entities.Incident
			if err := tx.Where("monitor_assertion_id = ? AND resolved = ?", incident.MonitorAssertionID, false).
				First(&existing).Error; err != nil {
				return err
			}

			if err := tx.Model(&existing).Update("updated_at", time.Now()).Error; err != nil {
				return err
			}

			*incident = existing
		} else {
			created = true
		}

		return tx.Create(&entities.IncidentOccurrence{IncidentID: incident.ID}).Error
	})

	return created, err
}

func (r *RepositoryImpl) Create(ctx context.Context, incidents *[]entities.Incident) error {
	return r.db.WithContext(ctx).Create(incidents).Error
}

func (r *RepositoryImpl) UpdateFields(ctx context.Context, teamID, id uint, fields map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&entities.Incident{}).
		Where("id = ? AND team_id = ?", id, teamID).
		Updates(fields)
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return ErrUnresolvedIncidentExists
		}

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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == postgres.ErrDuplicateEntry.Code
}
