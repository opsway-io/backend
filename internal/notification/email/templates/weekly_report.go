package templates

import _ "embed"

//go:embed weekly_report.hbs
var weeklyReportTemplate string

type WeeklyReportTemplate struct {
	TeamName         string
	StartDate        string
	EndDate          string
	DashboardURL     string
	TotalMonitors    int
	MonitorsUp       int
	MonitorsDown     int
	TotalIncidents   int
	ResolvedIncidents int
	AverageUptime    string
}

func (t *WeeklyReportTemplate) Subject() string {
	return "📊 Weekly Opsway Report for " + t.TeamName
}

func (t *WeeklyReportTemplate) HTML() string {
	return renderTemplate(weeklyReportTemplate, t)
}
