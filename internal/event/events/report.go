package events

type ReportGenerateTask struct {
	ReportID   uint   `json:"reportId"`
	TeamID     uint   `json:"teamId"`
	ReportType string `json:"reportType"`
	Start      string `json:"start"`
	End        string `json:"end"`
}

func (e ReportGenerateTask) Name() string {
	return "ReportGenerateTask"
}
