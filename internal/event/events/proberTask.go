package events

import "github.com/opsway-io/backend/internal/entities"

type ProberTask struct {
	Monitor  *entities.Monitor `json:"monitor"`
	Location string            `json:"location"`
}

func (e ProberTask) Name() string {
	return "prober.tasks." + e.Location
}
