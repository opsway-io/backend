package templates

import (
	_ "embed"
	"fmt"
)

//go:embed performance_degradation.hbs
var performanceDegradationTemplate string

type PerformanceDegradationTemplate struct {
	BaseTemplate
	MonitorName    string
	CurrentLatency string
	Threshold      string
	DashboardURL   string
}

func (t *PerformanceDegradationTemplate) Subject() string {
	return fmt.Sprintf("Warning: Performance Degradation on %s", t.MonitorName)
}

func (t *PerformanceDegradationTemplate) HTML() string {
	return t.Render(performanceDegradationTemplate, map[string]any{
		"MonitorName":    t.MonitorName,
		"CurrentLatency": t.CurrentLatency,
		"Threshold":      t.Threshold,
		"DashboardURL":   t.DashboardURL,
	})
}

func (t *PerformanceDegradationTemplate) PlainText() string {
	return "Performance Degradation Alert for " + t.MonitorName
}
