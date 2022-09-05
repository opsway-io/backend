package models

import "github.com/opsway-io/backend/internal/monitor"

type Monitor struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Tags       []string `json:"tags"`
	SettingsID int      `json:"settingsId"`
	CreatedAt  int64    `json:"createdAt"`
	UpdatedAt  int64    `json:"updatedAt"`
}

type MonitorSettings struct {
	ID        int                 `json:"id"`
	Method    string              `json:"method"`
	URL       string              `json:"url"`
	Headers   map[string]string   `json:"headers"`
	Body      MonitorSettingsBody `json:"body"`
	Interval  int                 `json:"interval"`
	CreatedAt int64               `json:"createdAt"`
	UpdatedAt int64               `json:"updatedAt"`
}

type MonitorSettingsBody struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func MonitorToResponse(m monitor.Monitor) Monitor {
	return Monitor{
		ID:         m.ID,
		Name:       m.Name,
		Tags:       m.Tags,
		SettingsID: m.SettingsID,
		CreatedAt:  m.CreatedAt.Unix(),
		UpdatedAt:  m.UpdatedAt.Unix(),
	}
}

func MonitorSettingsToResponse(ms monitor.Settings) MonitorSettings {
	return MonitorSettings{
		ID:        ms.ID,
		Method:    ms.Method,
		URL:       ms.URL,
		Interval:  int(ms.Interval.Seconds()),
		CreatedAt: ms.CreatedAt.Unix(),
		UpdatedAt: ms.UpdatedAt.Unix(),
		// TODO: add headers and body
	}
}

func MonitorsToResponse(monitors []monitor.Monitor) []Monitor {
	res := make([]Monitor, len(monitors))

	for i, m := range monitors {
		res[i] = MonitorToResponse(m)
	}

	return res
}
