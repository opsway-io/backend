package entities

import (
	"testing"
	"time"
)

func TestHeartbeatStatusAt(t *testing.T) {
	created := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	h := Heartbeat{Interval: 5, Grace: 1, CreatedAt: created}

	if got := h.StatusAt(created.Add(6 * time.Minute)); got != HeartbeatStatusUp {
		t.Fatalf("expected UP within interval+grace before first ping, got %s", got)
	}
	if got := h.StatusAt(created.Add(6*time.Minute + time.Second)); got != HeartbeatStatusDown {
		t.Fatalf("expected DOWN after interval+grace without ping, got %s", got)
	}

	ping := created.Add(time.Hour)
	h.LastPingAt = &ping
	if got := h.StatusAt(ping.Add(5 * time.Minute)); got != HeartbeatStatusUp {
		t.Fatalf("expected UP after recent ping, got %s", got)
	}
	if got := h.StatusAt(ping.Add(7 * time.Minute)); got != HeartbeatStatusDown {
		t.Fatalf("expected DOWN after missed ping, got %s", got)
	}

	h.Paused = true
	if got := h.StatusAt(ping.Add(time.Hour)); got != HeartbeatStatusPaused {
		t.Fatalf("expected PAUSED, got %s", got)
	}
}

func TestMaintenanceStatusAndCoverage(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	m := Maintenance{Settings: MaintenanceSettings{StartAt: start, EndAt: start.Add(time.Hour)}}

	cases := map[time.Time]MaintenanceStatus{
		start.Add(-time.Second):  MaintenanceStatusScheduled,
		start:                    MaintenanceStatusActive,
		start.Add(time.Hour - 1): MaintenanceStatusActive,
		start.Add(time.Hour):     MaintenanceStatusEnded,
	}
	for at, want := range cases {
		if got := m.StatusAt(at); got != want {
			t.Fatalf("StatusAt(%s) = %s, want %s", at, got, want)
		}
	}

	if !m.AppliesToMonitor(42) {
		t.Fatal("window without monitors should apply to all monitors")
	}

	m.Monitors = []Monitor{{ID: 1}, {ID: 2}}
	if !m.AppliesToMonitor(2) || m.AppliesToMonitor(3) {
		t.Fatal("window with monitors should only apply to those monitors")
	}
}
