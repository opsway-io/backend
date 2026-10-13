package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/boomerang"
	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack"
)

const taskKind = "http-probe"

type Schedule interface {
	Add(ctx context.Context, monitor *entities.Monitor) error
	Remove(ctx context.Context, monitor *entities.Monitor) error
	On(ctx context.Context, location string, handler func(ctx context.Context, monitor *entities.Monitor)) error
}

type ScheduleImpl struct {
	bschedule boomerang.Schedule
}

func NewSchedule(redisClient *redis.Client) Schedule {
	return &ScheduleImpl{
		bschedule: boomerang.NewSchedule(redisClient),
	}
}

func (s *ScheduleImpl) Add(ctx context.Context, monitor *entities.Monitor) error {
	data, err := s.marshalMonitor(monitor)
	if err != nil {
		return err
	}

	locations := monitor.Settings.Locations
	if len(locations) == 0 {
		locations = []string{"global"}
	}

	for _, loc := range locations {
		t := boomerang.NewTask(fmt.Sprintf("%s:%s", taskKind, loc), fmt.Sprintf("%d", monitor.ID), data)
		if err := s.bschedule.Add(ctx, t, monitor.Settings.Frequency, time.Now()); err != nil {
			return err
		}
	}

	return nil
}

func (s *ScheduleImpl) Remove(ctx context.Context, monitor *entities.Monitor) error {
	locations := monitor.Settings.Locations
	if len(locations) == 0 {
		locations = []string{"global"}
	}

	for _, loc := range locations {
		if err := s.bschedule.Remove(ctx, fmt.Sprintf("%s:%s", taskKind, loc), fmt.Sprintf("%d", monitor.ID)); err != nil {
			if err.Error() != "task does not exist" { // Check underlying boomerang error
				return err
			}
		}
	}
	return nil
}

func (s *ScheduleImpl) On(ctx context.Context, location string, handler func(ctx context.Context, monitor *entities.Monitor)) error {
	return s.bschedule.On(ctx, fmt.Sprintf("%s:%s", taskKind, location), func(ctx context.Context, task *boomerang.Task) {
		monitor, err := s.unmarshalMonitor(task.Data)
		if err != nil {
			return
		}

		handler(ctx, monitor)
	})
}

func (s *ScheduleImpl) marshalMonitor(monitor *entities.Monitor) ([]byte, error) {
	return msgpack.Marshal(monitor)
}

func (s *ScheduleImpl) unmarshalMonitor(data []byte) (*entities.Monitor, error) {
	var monitor entities.Monitor
	if err := msgpack.Unmarshal(data, &monitor); err != nil {
		return nil, err
	}

	return &monitor, nil
}
