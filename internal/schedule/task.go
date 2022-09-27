package scheduler

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"
	"github.com/opsway-io/backend/internal/probes"
)

type TaskType string

const (
	ProbeTask TaskType = "probe:http"
)

type TaskPayload struct {
	ID      int
	Payload map[string]string
}

func HandleTask(serv *probes.Service) asynq.HandlerFunc {
	fn := func(ctx context.Context, t *asynq.Task) error {
		var p TaskPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}

		log.Printf(" [*] Probe %s", p.Payload["URL"])
		return nil
	}

	return asynq.HandlerFunc(fn)
}

// type ProbeTaskPayload struct {
// 	TaskPayload
// 	URL string
// }
