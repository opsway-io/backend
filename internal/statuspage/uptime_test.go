package statuspage

import (
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/opsway-io/backend/internal/entities"
)

func TestBuildCheckStats(t *testing.T) {
	now := time.Date(2026, 10, 12, 15, 0, 0, 0, time.UTC)
	start := truncateDay(now).AddDate(0, 0, -(UptimeDays - 1))

	stats := buildCheckStats(
		[]uint{1, 2},
		[]latestRow{{MonitorID: 1, LatestOK: 0}},
		[]dailyRow{
			{MonitorID: 1, Day: truncateDay(now), Total: 4, OK: 3},
			{MonitorID: 1, Day: truncateDay(now).AddDate(0, 0, -1), Total: 4, OK: 4},
			{MonitorID: 1, Day: start.AddDate(0, 0, -1), Total: 10, OK: 0}, // out of range
		},
		start,
	)

	s1 := stats[1]
	if !s1.hasLatest || s1.latestOK {
		t.Fatalf("expected monitor 1 latest check failing, got %+v", s1)
	}
	if len(s1.daily) != UptimeDays || s1.daily[UptimeDays-1] != 75 || s1.daily[UptimeDays-2] != 100 || s1.daily[0] != -1 {
		t.Fatalf("unexpected daily uptimes: last=%v prev=%v first=%v", s1.daily[UptimeDays-1], s1.daily[UptimeDays-2], s1.daily[0])
	}
	if s1.uptime == nil || *s1.uptime != 87.5 {
		t.Fatalf("expected 87.5%% uptime, got %v", s1.uptime)
	}

	s2 := stats[2]
	if s2.hasLatest || s2.uptime != nil || s2.daily[UptimeDays-1] != -1 {
		t.Fatalf("expected no data for monitor 2, got %+v", s2)
	}
}

func TestAllMonitorIDsDedup(t *testing.T) {
	p := entities.StatusPage{
		MonitorIDs: pq.Int64Array{3, 1},
		Groups: []entities.StatusPageGroup{
			{MonitorIDs: pq.Int64Array{1, 2}},
			{MonitorIDs: pq.Int64Array{4}},
		},
	}

	got := p.AllMonitorIDs()
	want := []uint{3, 1, 2, 4}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestOverallStatus(t *testing.T) {
	v := PublicView{Monitors: []PublicMonitor{{Status: MonitorStatusOperational}}}
	if v.OverallStatus() != OverallStatusOperational {
		t.Fatal("expected operational")
	}
	v.ActiveMaintenance = []PublicMaintenance{{}}
	if v.OverallStatus() != OverallStatusMaintenance {
		t.Fatal("expected maintenance")
	}
	v.Monitors[0].Status = MonitorStatusOutage
	if v.OverallStatus() != OverallStatusOutage {
		t.Fatal("expected outage")
	}
}
