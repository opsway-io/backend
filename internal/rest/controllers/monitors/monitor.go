package monitors

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

/*
	Shared structs
*/

type Monitor struct {
	ID         uint               `json:"id"`
	State      string             `json:"state" validate:"required,monitorState"`
	Name       string             `json:"name" validate:"required,max=255"`
	Settings   MonitorSettings    `json:"settings" validate:"required,dive"`
	Assertions []MonitorAssertion `json:"assertions" validate:"required,monitorAssertions"`
	CreatedAt  time.Time          `json:"createdAt"`
	UpdatedAt  time.Time          `json:"updatedAt"`
}

type MonitorSettings struct {
	Method           string                  `json:"method" validate:"required,monitorMethod"`
	URL              string                  `json:"url" validate:"required,url"`
	FrequencySeconds uint64                  `json:"frequencySeconds" validate:"required,monitorFrequency"`
	Headers          []MonitorSettingsHeader `json:"headers" validate:"required,dive,required,max=255"`
	Body             MonitorSettingsBody     `json:"body" validate:"required,dive"`
	TLS              MonitorSettingsTLS      `json:"tls" validate:"required,dive"`
	Locations        []string                `json:"locations" validate:"omitempty,dive,required,max=255"`
}

type MonitorAssertion struct {
	Source   string `json:"source"`
	Property string `json:"property"`
	Operator string `json:"operator"`
	Target   string `json:"target"`
}

type MonitorSettingsHeader struct {
	Key   string `json:"key" validate:"required,max=255"`
	Value string `json:"value" validate:"max=255"`
}

type MonitorSettingsBody struct {
	Type    string  `json:"type" validate:"required,monitorBodyType"`
	Content *string `json:"content" validate:"omitempty,max=1048576"` // Max 1 MB
}

type MonitorSettingsTLS struct {
	Enabled                 bool  `json:"enabled"`
	VerifyHostname          *bool `json:"verifyHostname"`
	CheckExpiration         *bool `json:"checkExpiration"`
	ExpirationThresholdDays *uint `json:"expirationThresholdDays"`
}

/*
	Handlers
*/

type GetMonitorsRequest struct {
	TeamID uint    `param:"teamId" validate:"required,numeric,gte=0"`
	Offset *int    `query:"offset" validate:"numeric,gte=0" default:"0"`
	Limit  *int    `query:"limit" validate:"numeric,gt=0" default:"10"`
	Query  *string `query:"query" validate:"omitempty"`
}

type GetMonitorsResponse struct {
	Monitors   []GetMonitorsResponseMonitor `json:"monitors"`
	TotalCount int                          `json:"totalCount"`
}

type GetMonitorsResponseMonitor struct {
	Monitor
	Stats GetMonitorsResponseMonitorStats `json:"stats"`
}

type GetMonitorsResponseMonitorStats struct {
	UptimePercentage     float64   `json:"uptimePercentage"`
	AverageResponseTimes []float64 `json:"averageResponseTimes"`
	P99                  uint      `json:"p99"`
	P95                  uint      `json:"p95"`
}

func (h *Handlers) GetMonitors(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	monitors, err := h.MonitorService.GetMonitorsAndSettingsByTeamID(c.Request().Context(), req.TeamID, req.Offset, req.Limit, req.Query)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	monitorStats, err := h.CheckService.GetMonitorOverviewsByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitor overviews")

		return echo.ErrInternalServerError
	}

	resp, err := newGetMonitorsResponse(monitors, monitorStats)
	if err != nil {
		c.Log.WithError(err).Error("failed to create GetMonitorsResponse")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

func newGetMonitorsResponse(monitors *[]monitor.MonitorWithTotalCount, stats *[]check.MonitorOverviews) (*GetMonitorsResponse, error) {
	res := make([]GetMonitorsResponseMonitor, len(*monitors))

	monitorStatsMap := make(map[uint]check.MonitorOverviews, len(*stats))
	for _, m := range *stats {
		monitorStatsMap[m.MonitorID] = m
	}

	for i, m := range *monitors {
		headers := make([]MonitorSettingsHeader, len(m.Settings.Headers))
		for j, h := range m.Settings.Headers {
			headers[j] = MonitorSettingsHeader{
				Key:   h.Key,
				Value: h.Value,
			}
		}

		assertions := make([]MonitorAssertion, len(m.Assertions))
		for j, a := range m.Assertions {
			assertions[j] = MonitorAssertion{
				Source:   a.Source,
				Operator: a.Operator,
				Target:   a.Target,
				Property: a.Property,
			}
		}

		res[i] = GetMonitorsResponseMonitor{
			Monitor: Monitor{
				ID:        m.ID,
				State:     m.GetStateString(),
				Name:      m.Name,
				CreatedAt: m.CreatedAt,
				UpdatedAt: m.UpdatedAt,
				Settings: MonitorSettings{
					Method:           m.Settings.Method,
					URL:              m.Settings.URL,
					FrequencySeconds: m.Settings.GetFrequencySeconds(),
					Headers:          headers,
					Body: MonitorSettingsBody{
						Type:    m.Settings.Body.Type,
						Content: m.Settings.Body.GetContentString(),
					},
					TLS: MonitorSettingsTLS{
						Enabled:                 m.Settings.TLS.Enabled,
						VerifyHostname:          m.Settings.TLS.VerifyHostname,
						CheckExpiration:         m.Settings.TLS.CheckExpiration,
						ExpirationThresholdDays: m.Settings.TLS.ExpirationThresholdDays,
					},
					Locations: []string{}, // TODO: Implement
				},
				Assertions: assertions,
			},
		}

		stat, ok := monitorStatsMap[m.ID]
		if ok {
			res[i].Stats = GetMonitorsResponseMonitorStats{
				UptimePercentage:     0, // TODO: Implement
				AverageResponseTimes: stat.Stats,
				P99:                  uint(stat.P99),
				P95:                  uint(stat.P95),
			}
		}
	}

	totalCount := 0
	if len(*monitors) > 0 {
		totalCount = (*monitors)[0].TotalCount
	}

	return &GetMonitorsResponse{
		Monitors:   res,
		TotalCount: totalCount,
	}, nil
}

type GetMonitorRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
}

type GetMonitorResponse struct {
	Monitor
	Stats GetMonitorResponseStats `json:"stats"`
}

type GetMonitorResponseStats struct {
	UptimePercentage    float64 `json:"uptimePercentage"`
	AverageResponseTime float64 `json:"averageResponseTime"`
}

func (h *Handlers) GetMonitor(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorRequest")

		return echo.ErrBadRequest
	}
	m, err := h.MonitorService.GetMonitorAndSettingsByTeamIDAndID(c.Request().Context(), req.TeamID, req.MonitorID)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to get monitor")

		return echo.ErrInternalServerError
	}

	stats, err := h.CheckService.GetMonitorStatsByMonitorID(c.Request().Context(), m.ID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitor stats")

		return echo.ErrInternalServerError
	}

	resp, err := newGetMonitorResponse(m, stats)
	if err != nil {
		c.Log.WithError(err).Error("failed to create GetMonitorResponse")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

func newGetMonitorResponse(m *entities.Monitor, stats *check.MonitorStats) (*GetMonitorResponse, error) {
	headers := make([]MonitorSettingsHeader, len(m.Settings.Headers))
	for j, h := range m.Settings.Headers {
		headers[j] = MonitorSettingsHeader{
			Key:   h.Key,
			Value: h.Value,
		}
	}

	assertions := make([]MonitorAssertion, len(m.Assertions))
	for j, a := range m.Assertions {
		assertions[j] = MonitorAssertion{
			Source:   a.Source,
			Operator: a.Operator,
			Target:   a.Target,
			Property: a.Property,
		}
	}

	resp := GetMonitorResponse{
		Monitor: Monitor{
			ID:        m.ID,
			State:     m.GetStateString(),
			Name:      m.Name,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			Settings: MonitorSettings{
				Method:           m.Settings.Method,
				URL:              m.Settings.URL,
				FrequencySeconds: m.Settings.GetFrequencySeconds(),
				Headers:          headers,
				Body: MonitorSettingsBody{
					Type:    m.Settings.Body.Type,
					Content: m.Settings.Body.GetContentString(),
				},
				TLS: MonitorSettingsTLS{
					Enabled:                 m.Settings.TLS.Enabled,
					VerifyHostname:          m.Settings.TLS.VerifyHostname,
					CheckExpiration:         m.Settings.TLS.CheckExpiration,
					ExpirationThresholdDays: m.Settings.TLS.ExpirationThresholdDays,
				},
				Locations: []string{}, // TODO: Implement
			},
			Assertions: assertions,
		},
		Stats: GetMonitorResponseStats{
			UptimePercentage:    float64(stats.UptimePercentage),    // TODO: Implement
			AverageResponseTime: float64(stats.AverageResponseTime), // TODO: Implement
		},
	}

	return &resp, nil
}

type DeleteMonitorRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteMonitor(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteMonitorRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	if err := h.MonitorService.Delete(
		ctx,
		req.TeamID,
		req.MonitorID,
	); err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to delete monitor")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PostMonitorRequest struct {
	TeamID     uint               `param:"teamId" validate:"required,numeric,gte=0"`
	Name       string             `json:"name" validate:"required,max=255"`
	Settings   MonitorSettings    `json:"settings" validate:"required,dive"`
	Assertions []MonitorAssertion `json:"assertions" validate:"required,dive"`
}

func (h *Handlers) PostMonitor(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostMonitorRequest")

		return echo.ErrBadRequest
	}

	headers := make([]entities.MonitorSettingsHeader, len(req.Settings.Headers))
	for j, h := range req.Settings.Headers {
		headers[j] = entities.MonitorSettingsHeader{
			Key:   h.Key,
			Value: h.Value,
		}
	}

	assertions := make([]entities.MonitorAssertion, len(req.Assertions))
	for j, a := range req.Assertions {
		assertions[j] = entities.MonitorAssertion{
			Source:   a.Source,
			Operator: a.Operator,
			Target:   a.Target,
			Property: a.Property,
		}
	}

	m := &entities.Monitor{
		TeamID: req.TeamID,
		Name:   req.Name,
		Settings: entities.MonitorSettings{
			Method:  req.Settings.Method,
			URL:     req.Settings.URL,
			Headers: headers,
			Body: entities.MonitorSettingsBody{
				Type: req.Settings.Body.Type,
			},
			TLS: entities.MonitorSettingsTLS{
				Enabled:                 req.Settings.TLS.Enabled,
				VerifyHostname:          req.Settings.TLS.VerifyHostname,
				CheckExpiration:         req.Settings.TLS.CheckExpiration,
				ExpirationThresholdDays: req.Settings.TLS.ExpirationThresholdDays,
			},
			// TODO: Implement
			// Locations: req.Settings.Locations,
		},
		Assertions: assertions,
	}

	m.Settings.SetFrequencySeconds(req.Settings.FrequencySeconds)
	m.Settings.Body.SetContentString(req.Settings.Body.Content)

	if err := h.MonitorService.Create(c.Request().Context(), m); err != nil {
		c.Log.WithError(err).Error("failed to create monitor")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusCreated)
}

type PutMonitorRequest struct {
	TeamID     uint               `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID  uint               `param:"monitorId" validate:"required,numeric,gte=0"`
	Name       string             `json:"name" validate:"required,max=255"`
	State      string             `json:"state" validate:"required,monitorState"`
	Settings   MonitorSettings    `json:"settings" validate:"required,dive"`
	Assertions []MonitorAssertion `json:"assertions" validate:"required,dive"`
}

func (h *Handlers) PutMonitor(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[PutMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutMonitorRequest")

		return echo.ErrBadRequest
	}

	headers := make([]entities.MonitorSettingsHeader, len(req.Settings.Headers))
	for j, h := range req.Settings.Headers {
		headers[j] = entities.MonitorSettingsHeader{
			Key:   h.Key,
			Value: h.Value,
		}
	}

	assertions := make([]entities.MonitorAssertion, len(req.Assertions))
	for j, a := range req.Assertions {
		assertions[j] = entities.MonitorAssertion{
			Source:   a.Source,
			Operator: a.Operator,
			Target:   a.Target,
			Property: a.Property,
		}
	}

	m := &entities.Monitor{
		TeamID: req.TeamID,
		Name:   req.Name,
		Settings: entities.MonitorSettings{
			Method:  req.Settings.Method,
			URL:     req.Settings.URL,
			Headers: headers,
			Body: entities.MonitorSettingsBody{
				Type: req.Settings.Body.Type,
			},
			TLS: entities.MonitorSettingsTLS{
				Enabled:                 req.Settings.TLS.Enabled,
				VerifyHostname:          req.Settings.TLS.VerifyHostname,
				CheckExpiration:         req.Settings.TLS.CheckExpiration,
				ExpirationThresholdDays: req.Settings.TLS.ExpirationThresholdDays,
			},
			// TODO: Implement
			// Locations: req.Settings.Locations,
		},
		Assertions: assertions,
	}

	m.SetStateString(req.State)
	m.Settings.SetFrequencySeconds(req.Settings.FrequencySeconds)
	m.Settings.Body.SetContentString(req.Settings.Body.Content)

	if err := h.MonitorService.Update(
		ctx,
		req.TeamID,
		req.MonitorID,
		m,
	); err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to update monitor")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PutMonitorStateRequest struct {
	TeamID    uint   `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint   `param:"monitorId" validate:"required,numeric,gte=0"`
	State     string `json:"state" validate:"required,monitorState"`
}

func (h *Handlers) PutMonitorState(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[PutMonitorStateRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutMonitorStateRequest")

		return echo.ErrBadRequest
	}

	stateEnum := entities.GetMonitorStateEnumFromString(req.State)

	if err := h.MonitorService.SetState(
		ctx,
		req.TeamID,
		req.MonitorID,
		stateEnum,
	); err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to set monitor state")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}
