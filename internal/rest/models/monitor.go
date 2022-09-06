package models

import "github.com/opsway-io/backend/internal/monitor"

type Monitor struct {
	ID         int      `json:"id" validate:"required,numeric,gte=0"`
	Name       string   `json:"name" validate:"required,min=1,max=255"`
	Tags       []string `json:"tags" validate:"required,min=1,max=10,dive,min=1,max=255"`
	SettingsID int      `json:"settingsId" validate:"required,numeric,gte=0"`
	CreatedAt  int64    `json:"createdAt"`
	UpdatedAt  int64    `json:"updatedAt"`
}

type MonitorSettings struct {
	ID        int                 `json:"id" validate:"required,numeric,gte=0"`
	Method    string              `json:"method" validate:"required,oneof=GET POST PUT PATCH DELETE"`
	URL       string              `json:"url" validate:"required,url"`
	Headers   map[string]string   `json:"headers" validate:"max=50,dive,min=1,max=255"`
	Body      MonitorSettingsBody `json:"body"`
	Interval  int                 `json:"interval" validate:"required,numeric,gte=1"`
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

func RequestToMonitor(req Monitor) monitor.Monitor {
	return monitor.Monitor{
		ID:   req.ID,
		Name: req.Name,
		Tags: req.Tags,
	}
}
