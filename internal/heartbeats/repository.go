package heartbeats

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Repository interface {
	GetAllByTeamID(ctx context.Context, teamID uint) ([]entities.Heartbeat, error)
	GetByIDAndTeamID(ctx context.Context, teamID uint, heartbeatID uint) (*entities.Heartbeat, error)
	Create(ctx context.Context, heartbeat *entities.Heartbeat) error
	Update(ctx context.Context, heartbeat *entities.Heartbeat) error
	Delete(ctx context.Context, heartbeat *entities.Heartbeat) error
	Ping(ctx context.Context, heartbeatID uint) error
	GetExpiredHeartbeats(ctx context.Context) ([]entities.Heartbeat, error)
	MarkAsDown(ctx context.Context, heartbeatID uint) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
	}
}

func (r *RepositoryImpl) GetAllByTeamID(ctx context.Context, teamID uint) ([]entities.Heartbeat, error) {
	var heartbeats []entities.Heartbeat
	err := r.db.WithContext(ctx).Where("team_id = ?", teamID).Find(&heartbeats).Error
	return heartbeats, err
}

func (r *RepositoryImpl) GetByIDAndTeamID(ctx context.Context, teamID uint, heartbeatID uint) (*entities.Heartbeat, error) {
	var heartbeat entities.Heartbeat
	err := r.db.WithContext(ctx).Where("team_id = ? AND id = ?", teamID, heartbeatID).First(&heartbeat).Error
	return &heartbeat, err
}

func (r *RepositoryImpl) Create(ctx context.Context, heartbeat *entities.Heartbeat) error {
	return r.db.WithContext(ctx).Create(heartbeat).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, heartbeat *entities.Heartbeat) error {
	return r.db.WithContext(ctx).Save(heartbeat).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, heartbeat *entities.Heartbeat) error {
	return r.db.WithContext(ctx).Delete(heartbeat).Error
}

func (r *RepositoryImpl) Ping(ctx context.Context, heartbeatID uint) error {
	return r.db.WithContext(ctx).Model(&entities.Heartbeat{}).Where("id = ?", heartbeatID).Updates(map[string]interface{}{
		"last_ping": gorm.Expr("NOW()"),
		"status":    entities.HeartbeatStatusUp,
	}).Error
}

func (r *RepositoryImpl) GetExpiredHeartbeats(ctx context.Context) ([]entities.Heartbeat, error) {
	var heartbeats []entities.Heartbeat
	// last_ping + interval + grace < NOW()
	// Using PostgreSQL syntax for interval arithmetic
	err := r.db.WithContext(ctx).
		Where("status = ?", entities.HeartbeatStatusUp).
		Where("last_ping + make_interval(secs => (interval + grace) / 1000000000) < NOW()").
		Find(&heartbeats).Error
	return heartbeats, err
}

func (r *RepositoryImpl) MarkAsDown(ctx context.Context, heartbeatID uint) error {
	return r.db.WithContext(ctx).Model(&entities.Heartbeat{}).Where("id = ?", heartbeatID).Updates(map[string]interface{}{
		"status": entities.HeartbeatStatusDown,
	}).Error
}
