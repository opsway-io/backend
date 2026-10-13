package heartbeat

import (
	"context"
	"errors"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("heartbeat not found")

type Service interface {
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Heartbeat, error)
	GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.Heartbeat, error)
	Create(ctx context.Context, heartbeat *entities.Heartbeat) error
	Update(ctx context.Context, heartbeat *entities.Heartbeat) error
	Delete(ctx context.Context, teamID, id uint) error
	// Ping records a ping for the heartbeat and returns the updated heartbeat.
	Ping(ctx context.Context, teamID, id uint) (*entities.Heartbeat, error)
	// MarkOverdue flags heartbeats whose interval+grace elapsed without a ping as DOWN
	// and returns the heartbeats that just transitioned.
	MarkOverdue(ctx context.Context) ([]entities.Heartbeat, error)
}

type ServiceImpl struct {
	db  *gorm.DB
	now func() time.Time
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{db: db, now: time.Now}
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Heartbeat, error) {
	var heartbeats []entities.Heartbeat
	if err := s.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("name ASC, id ASC").
		Find(&heartbeats).Error; err != nil {
		return nil, err
	}

	return &heartbeats, nil
}

func (s *ServiceImpl) GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.Heartbeat, error) {
	var heartbeat entities.Heartbeat
	if err := s.db.WithContext(ctx).
		Where("team_id = ? AND id = ?", teamID, id).
		First(&heartbeat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &heartbeat, nil
}

func (s *ServiceImpl) Create(ctx context.Context, heartbeat *entities.Heartbeat) error {
	return s.db.WithContext(ctx).Create(heartbeat).Error
}

func (s *ServiceImpl) Update(ctx context.Context, heartbeat *entities.Heartbeat) error {
	result := s.db.WithContext(ctx).
		Model(&entities.Heartbeat{}).
		Where("team_id = ? AND id = ?", heartbeat.TeamID, heartbeat.ID).
		Updates(map[string]any{
			"name":       heartbeat.Name,
			"interval":   heartbeat.Interval,
			"grace":      heartbeat.Grace,
			"updated_at": s.now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, teamID, id uint) error {
	result := s.db.WithContext(ctx).
		Where("team_id = ? AND id = ?", teamID, id).
		Delete(&entities.Heartbeat{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *ServiceImpl) Ping(ctx context.Context, teamID, id uint) (*entities.Heartbeat, error) {
	now := s.now()

	result := s.db.WithContext(ctx).
		Model(&entities.Heartbeat{}).
		Where("team_id = ? AND id = ?", teamID, id).
		UpdateColumns(map[string]any{"last_ping_at": now, "down_since": nil})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return s.GetByTeamIDAndID(ctx, teamID, id)
}

func (s *ServiceImpl) MarkOverdue(ctx context.Context) ([]entities.Heartbeat, error) {
	now := s.now()

	var overdue []entities.Heartbeat
	err := s.db.WithContext(ctx).
		Model(&overdue).
		Clauses(clause.Returning{}).
		Where("paused = false AND down_since IS NULL").
		Where(`COALESCE(last_ping_at, created_at) + make_interval(mins => ("interval" + grace)::int) < ?`, now).
		UpdateColumn("down_since", now).Error
	if err != nil {
		return nil, err
	}

	return overdue, nil
}

// RunChecker periodically marks overdue heartbeats as DOWN until ctx is cancelled.
func RunChecker(ctx context.Context, svc Service, every time.Duration, logger *logrus.Entry) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		down, err := svc.MarkOverdue(ctx)
		if err != nil {
			logger.WithError(err).Error("failed to check heartbeats")
		}

		for _, h := range down {
			logger.WithFields(logrus.Fields{
				"heartbeat_id": h.ID,
				"team_id":      h.TeamID,
				"name":         h.Name,
			}).Info("heartbeat is DOWN: no ping within interval + grace")
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
