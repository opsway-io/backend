package events

type TrainForecasterEvent struct {
	TeamID    uint `json:"team_id"`
	MonitorID uint `json:"monitor_id"`
}

func (e TrainForecasterEvent) Name() string {
	return "TrainForecasterTask"
}
