package schedule

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
)

type Schedule interface {
	Add(ctx context.Context, internval time.Duration, taskType TaskType, taskPayload TaskPayload) (string, error)
	Remove(ctx context.Context, EntryID string) (err error)
	Consume(ctx context.Context, handlers map[string]asynq.HandlerFunc) error
}
