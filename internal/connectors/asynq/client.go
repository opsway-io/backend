package asynq

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Addr string `required:"true"`
}

func handleEnqueueError(task *asynq.Task, opts []asynq.Option, err error) {
	logrus.Error(err)
}

func NewServer(ctx context.Context, conf Config) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: conf.Addr},
		asynq.Config{
			Concurrency: 10,
		},
	)
}

func NewScheduler(ctx context.Context, conf Config) *asynq.Scheduler {
	redisConnOpt := asynq.RedisClientOpt{Addr: conf.Addr}
	return asynq.NewScheduler(redisConnOpt, &asynq.SchedulerOpts{
		EnqueueErrorHandler: handleEnqueueError,
	})
}
