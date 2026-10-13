package templates

import _ "embed"

//go:embed weekly_report.hbs
var weeklyReportTemplate string

type WeeklyReportTemplate struct {
	BaseTemplate
	TeamName          string
	StartDate         string
	EndDate           string
	DashboardURL      string
	TotalMonitors     int
	MonitorsUp        int
	MonitorsDown      int
	TotalIncidents    int
	ResolvedIncidents int
	AverageUptime     string
}

func (t *WeeklyReportTemplate) Subject() string {
	return "📊 Weekly Opsway Report for " + t.TeamName
}

func (t *WeeklyReportTemplate) HTML() string {
	return t.Render(weeklyReportTemplate, map[string]any{
		"TeamName":          t.TeamName,
		"StartDate":         t.StartDate,
		"EndDate":           t.EndDate,
		"DashboardURL":      t.DashboardURL,
		"TotalMonitors":     t.TotalMonitors,
		"MonitorsUp":        t.MonitorsUp,
		"MonitorsDown":      t.MonitorsDown,
		"TotalIncidents":    t.TotalIncidents,
		"ResolvedIncidents": t.ResolvedIncidents,
		"AverageUptime":     t.AverageUptime,
	})
}

func (t *WeeklyReportTemplate) PlainText() string {
	return "Weekly Report"
}
