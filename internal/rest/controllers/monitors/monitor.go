package monitors

import (
	"errors"
	"fmt"
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
	Steps      []MonitorStep      `json:"steps" validate:"required,dive"`
	CreatedAt  time.Time          `json:"createdAt"`
	UpdatedAt  time.Time          `json:"updatedAt"`
}

type MonitorStep struct {
	Name       string             `json:"name" validate:"required,max=255"`
	Method     string             `json:"method" validate:"required,monitorMethod"`
	URL        string             `json:"url" validate:"required,url"`
	Headers    []MonitorSettingsHeader `json:"headers" validate:"dive"`
	Body       MonitorSettingsBody     `json:"body" validate:"dive"`
	Assertions []MonitorAssertion `json:"assertions" validate:"dive,monitorAssertions"`
	Variables  []MonitorVariable  `json:"variables" validate:"dive"`
}

type MonitorSettings struct {
	FrequencySeconds uint64                  `json:"frequencySeconds" validate:"required,numeric,gte=10"`
	Auth             MonitorSettingsAuth     `json:"auth" validate:"required"`
	TLS              MonitorSettingsTLS      `json:"tls" validate:"required"`
	Locations        []string                `json:"locations" validate:"required,dive,location"`
}

type MonitorSettingsTeardown struct {
	Enabled bool                `json:"enabled"`
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Body    MonitorSettingsBody `json:"body"`
}

type MonitorVariable struct {
	Name     string `json:"name" validate:"required"`
	Source   string `json:"source" validate:"required"`
	Property string `json:"property" validate:"required"`
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

type MonitorSettingsAuth struct {
	Method       string `json:"method" validate:"required"`
	TokenURL     string `json:"tokenUrl" validate:"omitempty,url"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Username     string `json:"username"`
	Password     string `json:"password"`
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
	AverageResponseTime  float64   `json:"averageResponseTime"`
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
		locations := m.Settings.Locations
		if locations == nil {
			locations = []string{}
		}

		steps := make([]MonitorStep, len(m.Steps))
		for j, s := range m.Steps {
			headers := make([]MonitorSettingsHeader, len(s.Headers))
			for k, h := range s.Headers {
				headers[k] = MonitorSettingsHeader{
					Key:   h.Key,
					Value: h.Value,
				}
			}

			assertions := make([]MonitorAssertion, len(s.Assertions))
			for k, a := range s.Assertions {
				assertions[k] = MonitorAssertion{
					Source:   a.Source,
					Operator: a.Operator,
					Target:   a.Target,
					Property: a.Property,
				}
			}

			variables := make([]MonitorVariable, len(s.Variables))
			for k, v := range s.Variables {
				variables[k] = MonitorVariable{
					Name:     v.Name,
					Source:   v.Source,
					Property: v.Property,
				}
			}
			steps[j] = MonitorStep{
				Name:       s.Name,
				Method:     s.Method,
				URL:        s.URL,
				Headers:    headers,
				Body: MonitorSettingsBody{
					Type:    s.Body.Type,
					Content: s.Body.GetContentString(),
				},
				Assertions: assertions,
				Variables:  variables,
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
					FrequencySeconds: m.Settings.GetFrequencySeconds(),
					TLS: MonitorSettingsTLS{
						Enabled:                 m.Settings.TLS.Enabled,
						VerifyHostname:          m.Settings.TLS.VerifyHostname,
						CheckExpiration:         m.Settings.TLS.CheckExpiration,
						ExpirationThresholdDays: m.Settings.TLS.ExpirationThresholdDays,
					},
					Auth: MonitorSettingsAuth{
						Method:       m.Settings.Auth.Method,
						TokenURL:     m.Settings.Auth.TokenURL,
						ClientID:     m.Settings.Auth.ClientID,
						ClientSecret: m.Settings.Auth.ClientSecret,
						Username:     m.Settings.Auth.Username,
						Password:     m.Settings.Auth.Password,
					},
					Locations: locations,
				},
				Steps: steps,
			},
		}

		stat, ok := monitorStatsMap[m.ID]
		if ok {
			res[i].Stats = GetMonitorsResponseMonitorStats{
				UptimePercentage:     float64(stat.UptimePercentage),
				AverageResponseTime:  float64(stat.AverageResponseTime),
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
	locations := m.Settings.Locations
	if locations == nil {
		locations = []string{}
	}

	steps := make([]MonitorStep, len(m.Steps))
	for j, s := range m.Steps {
		headers := make([]MonitorSettingsHeader, len(s.Headers))
		for k, h := range s.Headers {
			headers[k] = MonitorSettingsHeader{
				Key:   h.Key,
				Value: h.Value,
			}
		}

		assertions := make([]MonitorAssertion, len(s.Assertions))
		for k, a := range s.Assertions {
			assertions[k] = MonitorAssertion{
				Source:   a.Source,
				Operator: a.Operator,
				Target:   a.Target,
				Property: a.Property,
			}
		}

		variables := make([]MonitorVariable, len(s.Variables))
		for k, v := range s.Variables {
			variables[k] = MonitorVariable{
				Name:     v.Name,
				Source:   v.Source,
				Property: v.Property,
			}
		}
		steps[j] = MonitorStep{
			Name:       s.Name,
			Method:     s.Method,
			URL:        s.URL,
			Headers:    headers,
			Body: MonitorSettingsBody{
				Type:    s.Body.Type,
				Content: s.Body.GetContentString(),
			},
			Assertions: assertions,
			Variables:  variables,
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
				FrequencySeconds: m.Settings.GetFrequencySeconds(),
				TLS: MonitorSettingsTLS{
					Enabled:                 m.Settings.TLS.Enabled,
					VerifyHostname:          m.Settings.TLS.VerifyHostname,
					CheckExpiration:         m.Settings.TLS.CheckExpiration,
					ExpirationThresholdDays: m.Settings.TLS.ExpirationThresholdDays,
				},
				Auth: MonitorSettingsAuth{
					Method:       m.Settings.Auth.Method,
					TokenURL:     m.Settings.Auth.TokenURL,
					ClientID:     m.Settings.Auth.ClientID,
					ClientSecret: m.Settings.Auth.ClientSecret,
					Username:     m.Settings.Auth.Username,
					Password:     m.Settings.Auth.Password,
				},
				Locations: locations,
			},
			Steps: steps,
		},
		Stats: GetMonitorResponseStats{
			UptimePercentage:    float64(stats.UptimePercentage),
			AverageResponseTime: float64(stats.AverageResponseTime),
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
	Steps      []MonitorStep      `json:"steps" validate:"required,dive"`
}

func (h *Handlers) PostMonitor(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostMonitorRequest")
		fmt.Println("BIND ERROR POST MONITOR:", err)
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()
	teamEntity, err := h.TeamService.GetByID(ctx, req.TeamID)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get team")
		return echo.ErrInternalServerError
	}

	limit := teamEntity.GetMonitorLimit()
	if limit != -1 {
		count, err := h.MonitorService.CountByTeamID(ctx, req.TeamID)
		if err != nil {
			c.Log.WithError(err).Debug("failed to get monitor count")
			return echo.ErrInternalServerError
		}
		if count >= int64(limit) {
			return echo.NewHTTPError(http.StatusPaymentRequired, "Payment Required")
		}
	}

	steps := make([]entities.MonitorStep, len(req.Steps))
	for i, s := range req.Steps {
		headers := make([]entities.MonitorStepHeader, len(s.Headers))
		for j, h := range s.Headers {
			headers[j] = entities.MonitorStepHeader{
				Key:   h.Key,
				Value: h.Value,
			}
		}

		assertions := make([]entities.MonitorAssertion, len(s.Assertions))
		for j, a := range s.Assertions {
			assertions[j] = entities.MonitorAssertion{
				Source:   a.Source,
				Operator: a.Operator,
				Target:   a.Target,
				Property: a.Property,
			}
		}

		variables := make([]entities.MonitorVariable, len(s.Variables))
		for j, v := range s.Variables {
			variables[j] = entities.MonitorVariable{
				Name:     v.Name,
				Source:   v.Source,
				Property: v.Property,
			}
		}
		
		body := entities.MonitorStepBody{
			Type: s.Body.Type,
		}
		body.SetContentString(s.Body.Content)

		steps[i] = entities.MonitorStep{
			Name:       s.Name,
			OrderIndex: i,
			Method:     s.Method,
			URL:        s.URL,
			Headers:    headers,
			Body:       body,
			Assertions: assertions,
			Variables:  variables,
		}
	}

	m := &entities.Monitor{
		TeamID: req.TeamID,
		Name:   req.Name,
		Settings: entities.MonitorSettings{
			TLS: entities.MonitorSettingsTLS{
				Enabled:                 req.Settings.TLS.Enabled,
				VerifyHostname:          req.Settings.TLS.VerifyHostname,
				CheckExpiration:         req.Settings.TLS.CheckExpiration,
				ExpirationThresholdDays: req.Settings.TLS.ExpirationThresholdDays,
			},
			Auth: entities.MonitorSettingsAuth{
				Method:       req.Settings.Auth.Method,
				TokenURL:     req.Settings.Auth.TokenURL,
				ClientID:     req.Settings.Auth.ClientID,
				ClientSecret: req.Settings.Auth.ClientSecret,
				Username:     req.Settings.Auth.Username,
				Password:     req.Settings.Auth.Password,
			},
			Locations: req.Settings.Locations,
		},
		Steps: steps,
	}

	m.Settings.SetFrequencySeconds(req.Settings.FrequencySeconds)

	if err := h.MonitorService.Create(c.Request().Context(), m); err != nil {
		c.Log.WithError(err).Error("failed to create monitor")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusCreated)
}

type GetMonitorIncidents struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

type GetMonitorIncidentsResponse struct {
	Monitors []MonitorWithIncidents `json:"monitors"`
}

type MonitorWithIncidents struct {
	ID        uint       `json:"id"`
	TeamID    uint       `json:"teamId"`
	State     string     `json:"state" validate:"required,monitorState"`
	Name      string     `json:"name" validate:"required,max=255"`
	Incidents []Incident `json:"incidents" validate:"required,incidents"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type Incident struct {
	ID                 uint      `json:"id"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	MonitorAssertionID *uint     `json:"monitorAssertionId"`
	Resolved           bool      `json:"resolved"`
	Acknowledged       bool      `json:"acknowledged"`
}

func (h *Handlers) GetMonitorIncidents(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorIncidents](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorIncidents")

		return echo.ErrBadRequest
	}

	monitors, err := h.MonitorService.GetMonitorsAndIncidentsByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitor incidents")
		return echo.ErrInternalServerError
	}

	// manuel filtering should be in query :P
	filteredMonitors := make([]entities.Monitor, 0, len(*monitors))
	for _, m := range *monitors {
		if len(m.Incidents) > 0 {
			filteredMonitors = append(filteredMonitors, m)
		}
	}

	resp, err := newGetMonitorWithIncidentsResponse(&filteredMonitors)
	if err != nil {
		c.Log.WithError(err).Error("failed to create GetMonitorIncidentsResponse")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

func newGetMonitorWithIncidentsResponse(monitors *[]entities.Monitor) (*GetMonitorIncidentsResponse, error) {
	res := make([]MonitorWithIncidents, len(*monitors))

	for i, m := range *monitors {
		monitorWithIncidents := MonitorWithIncidents{
			ID:        m.ID,
			TeamID:    m.TeamID,
			State:     m.GetStateString(),
			Name:      m.Name,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			Incidents: make([]Incident, len(m.Incidents)),
		}
		for j, incident := range m.Incidents {
			monitorWithIncidents.Incidents[j] = Incident{
				ID:                 incident.ID,
				CreatedAt:          incident.CreatedAt,
				UpdatedAt:          incident.UpdatedAt,
				MonitorAssertionID: incident.MonitorAssertionID,
				Resolved:           incident.Resolved,
				Acknowledged:       incident.Acknowledged,
			}
		}
		res[i] = monitorWithIncidents

	}
	return &GetMonitorIncidentsResponse{
		Monitors: res,
	}, nil
}

type PutMonitorRequest struct {
	TeamID     uint               `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID  uint               `param:"monitorId" validate:"required,numeric,gte=0"`
	Name       string             `json:"name" validate:"required,max=255"`
	State      string             `json:"state" validate:"required,monitorState"`
	Settings   MonitorSettings    `json:"settings" validate:"required,dive"`
	Steps      []MonitorStep      `json:"steps" validate:"required,dive"`
}

func (h *Handlers) PutMonitor(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[PutMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutMonitorRequest")

		return echo.ErrBadRequest
	}

	steps := make([]entities.MonitorStep, len(req.Steps))
	for i, s := range req.Steps {
		headers := make([]entities.MonitorStepHeader, len(s.Headers))
		for j, h := range s.Headers {
			headers[j] = entities.MonitorStepHeader{
				Key:   h.Key,
				Value: h.Value,
			}
		}

		assertions := make([]entities.MonitorAssertion, len(s.Assertions))
		for j, a := range s.Assertions {
			assertions[j] = entities.MonitorAssertion{
				Source:   a.Source,
				Operator: a.Operator,
				Target:   a.Target,
				Property: a.Property,
			}
		}

		variables := make([]entities.MonitorVariable, len(s.Variables))
		for j, v := range s.Variables {
			variables[j] = entities.MonitorVariable{
				Name:     v.Name,
				Source:   v.Source,
				Property: v.Property,
			}
		}

		body := entities.MonitorStepBody{
			Type: s.Body.Type,
		}
		body.SetContentString(s.Body.Content)
		
		steps[i] = entities.MonitorStep{
			Name:       s.Name,
			OrderIndex: i,
			Method:     s.Method,
			URL:        s.URL,
			Headers:    headers,
			Body:       body,
			Assertions: assertions,
			Variables:  variables,
		}
	}

	m := &entities.Monitor{
		TeamID: req.TeamID,
		Name:   req.Name,
		Settings: entities.MonitorSettings{
			TLS: entities.MonitorSettingsTLS{
				Enabled:                 req.Settings.TLS.Enabled,
				VerifyHostname:          req.Settings.TLS.VerifyHostname,
				CheckExpiration:         req.Settings.TLS.CheckExpiration,
				ExpirationThresholdDays: req.Settings.TLS.ExpirationThresholdDays,
			},
			Auth: entities.MonitorSettingsAuth{
				Method:       req.Settings.Auth.Method,
				TokenURL:     req.Settings.Auth.TokenURL,
				ClientID:     req.Settings.Auth.ClientID,
				ClientSecret: req.Settings.Auth.ClientSecret,
				Username:     req.Settings.Auth.Username,
				Password:     req.Settings.Auth.Password,
			},
			Locations: req.Settings.Locations,
		},
		Steps: steps,
	}

	m.SetStateString(req.State)
	m.Settings.SetFrequencySeconds(req.Settings.FrequencySeconds)

	if req.State == "ACTIVE" {
		activeMaintenances, err := h.MaintenanceService.GetActive(ctx, time.Now())
		if err != nil {
			c.Log.WithError(err).Error("failed to get active maintenances")
			return echo.ErrInternalServerError
		}

		for _, maintenance := range *activeMaintenances {
			if maintenance.TeamID == req.TeamID {
				appliesToAll := len(maintenance.Monitors) == 0
				appliesToThis := false
				if !appliesToAll {
					for _, m := range maintenance.Monitors {
						if m.ID == req.MonitorID {
							appliesToThis = true
							break
						}
					}
				}

				if appliesToAll || appliesToThis {
					return echo.NewHTTPError(http.StatusConflict, "Cannot resume monitor while an active maintenance window is ongoing for this monitor. Please complete or delete the maintenance first.")
				}
			}
		}
	}

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

	if req.State == "ACTIVE" {
		activeMaintenances, err := h.MaintenanceService.GetActive(ctx, time.Now())
		if err != nil {
			c.Log.WithError(err).Error("failed to get active maintenances")
			return echo.ErrInternalServerError
		}

		for _, maintenance := range *activeMaintenances {
			if maintenance.TeamID == req.TeamID {
				appliesToAll := len(maintenance.Monitors) == 0
				appliesToThis := false
				if !appliesToAll {
					for _, m := range maintenance.Monitors {
						if m.ID == req.MonitorID {
							appliesToThis = true
							break
						}
					}
				}

				if appliesToAll || appliesToThis {
					return echo.NewHTTPError(http.StatusConflict, "Cannot resume monitor while an active maintenance window is ongoing for this monitor. Please complete or delete the maintenance first.")
				}
			}
		}
	}

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
