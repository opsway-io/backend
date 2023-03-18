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
	Remove(ctx context.Context, monitorID uint) error
	On(ctx context.Context, handler func(ctx context.Context, monitor *entities.Monitor)) error
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

	t := boomerang.NewTask(taskKind, fmt.Sprintf("%d", monitor.ID), data)

	fmt.Println("monitor.Settings.Frequency", monitor.Settings.Frequency)

	return s.bschedule.Add(ctx, t, monitor.Settings.Frequency, time.Now())
}

func (s *ScheduleImpl) Remove(ctx context.Context, monitorID uint) error {
	return s.bschedule.Remove(ctx, taskKind, fmt.Sprintf("%d", monitorID))
}

func (s *ScheduleImpl) On(ctx context.Context, handler func(ctx context.Context, monitor *entities.Monitor)) error {
	return s.bschedule.On(ctx, taskKind, func(ctx context.Context, task *boomerang.Task) {
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
