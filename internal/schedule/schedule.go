package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

type Schedule interface {
	Add(ctx context.Context, internval time.Duration, taskType TaskType, taskPayload TaskPayload) (string, error)
	Remove(ctx context.Context, EntryID string) (err error)
	Consume(ctx context.Context, handlers map[string]func(context.Context, *asynq.Task) error) (err error)
}

type AsynqSchedule struct {
	Scheduler *asynq.Scheduler
	Server    *asynq.Server
}

func New(scheduler *asynq.Scheduler, server *asynq.Server) *AsynqSchedule {
	return &AsynqSchedule{Scheduler: scheduler, Server: server}
}

func (rs *AsynqSchedule) Add(ctx context.Context, internval time.Duration, taskType TaskType, taskPayload TaskPayload) (string, error) {
	logrus.Info("Publishing event")

	payload, err := json.Marshal(taskPayload)
	if err != nil {
		return "", err
	}

	task := asynq.NewTask(string(taskType), payload)

	return rs.Scheduler.Register(fmt.Sprintf("@every %s", internval.String()), task)
}

func (rs *AsynqSchedule) Remove(ctx context.Context, entryID string) (err error) {
	return rs.Scheduler.Unregister(entryID)
}

func (rs *AsynqSchedule) Consume(ctx context.Context, handlers map[TaskType]func(context.Context, *asynq.Task) error) error {
	mux := asynq.NewServeMux()
	for pattern, handler := range handlers {
		logrus.Info("Handling events of type", pattern)
		mux.HandleFunc(string(pattern), handler)
	}

	return rs.Server.Run(mux)
}
