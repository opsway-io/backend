package schedule

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/probes"
	httpProbe "github.com/opsway-io/backend/internal/probes/http"
	"github.com/sirupsen/logrus"
)

type TaskType string

const (
	ProbeTask TaskType = "probe:http"
)

type TaskPayload struct {
	ID      int
	Payload map[string]string
}

func HandleTask(serv probes.Service) asynq.HandlerFunc {
	fn := func(ctx context.Context, t *asynq.Task) error {
		var p TaskPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}

		res, err := httpProbe.Probe(http.MethodGet, p.Payload["URL"], nil, nil, time.Second*5)
		if err != nil {
			logrus.WithError(err).Fatal("error probing url")
		}

		serv.Create(ctx, &entities.HttpResult{MonitorID: p.ID, Body: res})
		log.Printf(" [*] Probe %s", p.Payload["URL"])
		log.Printf("Result: %v", res.Response)
		return nil
	}

	return asynq.HandlerFunc(fn)
}

// type ProbeTaskPayload struct {
// 	TaskPayload
// 	URL string
// }
