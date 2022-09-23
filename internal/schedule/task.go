package scheduler

const (
	ProbeTask = "probe:http"
)

type TaskPayload struct {
	ID      int
	Payload map[string]string
}

// type ProbeTaskPayload struct {
// 	TaskPayload
// 	URL string
// }
