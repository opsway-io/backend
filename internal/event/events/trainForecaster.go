package events

type TrainForecasterEvent struct {
	MonitorID uint `json:"monitor_id"`
}

func (e TrainForecasterEvent) Name() string {
	return "TrainForecasterTask"
}
