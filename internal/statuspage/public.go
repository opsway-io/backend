package statuspage

import (
	"context"
	"html"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/maintenance"
	"gorm.io/gorm"
)

const UptimeDays = 90

type MonitorStatus string

const (
	MonitorStatusOperational MonitorStatus = "OPERATIONAL"
	MonitorStatusOutage      MonitorStatus = "OUTAGE"
)

type OverallStatus string

const (
	OverallStatusOperational OverallStatus = "OPERATIONAL"
	OverallStatusOutage      OverallStatus = "OUTAGE"
	OverallStatusMaintenance OverallStatus = "MAINTENANCE"
)

type PublicMonitor struct {
	ID               uint          `json:"id"`
	Name             string        `json:"name"`
	Status           MonitorStatus `json:"status"`
	CreatedAt        time.Time     `json:"createdAt"`
	UptimePercentage *float64      `json:"uptimePercentage,omitempty"`
	// Daily uptime percentages, oldest first, the last element is today. -1 means no data.
	DailyUptimes []float64 `json:"dailyUptimes"`
}

type PublicIncident struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	MonitorID   *uint  `json:"monitorId"`
}

type PublicMaintenance struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartAt     time.Time `json:"startAt"`
	EndAt       time.Time `json:"endAt"`
}

type PublicGroup struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Order      int    `json:"order"`
	MonitorIDs []uint `json:"monitorIds"`
}

type PublicView struct {
	Monitors          []PublicMonitor     `json:"monitors"`
	Groups            []PublicGroup       `json:"groups"`
	ActiveIncidents   []PublicIncident    `json:"activeIncidents"`
	ActiveMaintenance []PublicMaintenance `json:"activeMaintenance"`
	MaintenanceEvents []PublicMaintenance `json:"maintenanceEvents"`
}

func (v *PublicView) OverallStatus() OverallStatus {
	if len(v.ActiveIncidents) > 0 {
		return OverallStatusOutage
	}
	for _, m := range v.Monitors {
		if m.Status == MonitorStatusOutage {
			return OverallStatusOutage
		}
	}
	if len(v.ActiveMaintenance) > 0 {
		return OverallStatusMaintenance
	}

	return OverallStatusOperational
}

func (s *ServiceImpl) GetOverallStatus(ctx context.Context, page *entities.StatusPage) (OverallStatus, error) {
	view, err := s.GetPublicView(ctx, page)
	if err != nil {
		return "", err
	}

	return view.OverallStatus(), nil
}

func (s *ServiceImpl) GetPublicView(ctx context.Context, page *entities.StatusPage) (*PublicView, error) {
	now := s.now()
	ids := page.AllMonitorIDs()

	view := &PublicView{
		Monitors:          []PublicMonitor{},
		Groups:            []PublicGroup{},
		ActiveIncidents:   []PublicIncident{},
		ActiveMaintenance: []PublicMaintenance{},
		MaintenanceEvents: []PublicMaintenance{},
	}

	// Monitors (deleted monitors silently drop out).
	monitorsByID := map[uint]entities.Monitor{}
	if len(ids) > 0 {
		var monitors []entities.Monitor
		if err := s.db.WithContext(ctx).
			Select("id", "team_id", "name", "created_at").
			Where("team_id = ? AND id IN ?", page.TeamID, ids).
			Find(&monitors).Error; err != nil {
			return nil, err
		}
		for _, m := range monitors {
			monitorsByID[m.ID] = m
		}
	}

	existing := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ok := monitorsByID[id]; ok {
			existing = append(existing, id)
		}
	}

	for _, g := range page.Groups {
		mids := []uint{}
		for _, id := range g.MonitorIDs {
			if _, ok := monitorsByID[uint(id)]; ok {
				mids = append(mids, uint(id))
			}
		}
		view.Groups = append(view.Groups, PublicGroup{ID: g.ID, Name: g.Name, Order: g.Order, MonitorIDs: mids})
	}

	// Maintenance windows relevant to the page (recent, active and upcoming).
	var windows []entities.Maintenance
	if err := s.db.WithContext(ctx).
		Preload("Settings").
		Preload("Monitors", func(db *gorm.DB) *gorm.DB { return db.Select("monitors.id") }).
		Joins("JOIN maintenance_settings ON maintenance_settings.maintenance_id = maintenance.id").
		Where("maintenance.team_id = ?", page.TeamID).
		Where("maintenance_settings.end_at >= ?", now.AddDate(0, 0, -UptimeDays)).
		Order("maintenance_settings.start_at ASC").
		Find(&windows).Error; err != nil {
		return nil, err
	}

	active := []entities.Maintenance{}
	for i := range windows {
		w := &windows[i]
		if !windowTouchesPage(w, existing) {
			continue
		}

		pm := PublicMaintenance{
			ID:          w.ID,
			Title:       w.Title,
			Description: deref(w.Description),
			StartAt:     w.Settings.StartAt,
			EndAt:       w.Settings.EndAt,
		}
		view.MaintenanceEvents = append(view.MaintenanceEvents, pm)

		if w.StatusAt(now) == entities.MaintenanceStatusActive {
			view.ActiveMaintenance = append(view.ActiveMaintenance, pm)
			active = append(active, *w)
		}
	}
	underMaintenance := maintenance.CoversMonitor(active)

	// Unresolved incidents for the page's monitors. Any unresolved incident
	// marks the monitor as an outage.
	incidentMonitors := map[uint]struct{}{}
	if len(existing) > 0 {
		var incidents []entities.Incident
		if err := s.db.WithContext(ctx).
			Where("team_id = ? AND monitor_id IN ? AND resolved = ?", page.TeamID, existing, false).
			Order("created_at DESC").
			Find(&incidents).Error; err != nil {
			return nil, err
		}
		for _, in := range incidents {
			incidentMonitors[in.MonitorID] = struct{}{}

			// Only incidents explicitly published to status pages are listed.
			if !in.IsStatusPageVisible {
				continue
			}

			monitorID := in.MonitorID
			view.ActiveIncidents = append(view.ActiveIncidents, PublicIncident{
				ID:          in.ID,
				Title:       in.Title,
				Description: deref(in.Description),
				MonitorID:   &monitorID,
			})
		}
	}

	stats, err := s.monitorCheckStats(ctx, page.TeamID, existing, now)
	if err != nil {
		return nil, err
	}

	for _, id := range existing {
		m := monitorsByID[id]
		st := stats[id]

		status := MonitorStatusOperational
		if _, ok := incidentMonitors[id]; ok {
			status = MonitorStatusOutage
		} else if st.hasLatest && !st.latestOK {
			status = MonitorStatusOutage
		}
		if underMaintenance(id) {
			status = MonitorStatusOperational
		}

		view.Monitors = append(view.Monitors, PublicMonitor{
			ID:               id,
			Name:             m.Name,
			Status:           status,
			CreatedAt:        m.CreatedAt,
			UptimePercentage: st.uptime,
			DailyUptimes:     st.daily,
		})
	}

	return view, nil
}

func windowTouchesPage(w *entities.Maintenance, monitorIDs []uint) bool {
	if len(w.Monitors) == 0 {
		return true
	}

	for _, id := range monitorIDs {
		if w.AppliesToMonitor(id) {
			return true
		}
	}

	return false
}

func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func htmlEscape(s string) string {
	return html.EscapeString(s)
}
