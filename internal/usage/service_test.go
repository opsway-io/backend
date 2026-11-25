package usage

import (
	"testing"
	"time"
)

func TestBuildQuotaItem(t *testing.T) {
	okItem := buildQuotaItem("monitors", "Active Monitors", 10, 50, "monitors", nil)
	if okItem.Status != "ok" || okItem.Percent != 20.0 {
		t.Fatalf("expected ok/20.0, got %s/%.1f", okItem.Status, okItem.Percent)
	}

	warnItem := buildQuotaItem("logs", "Log Ingestion", 42, 50, "GB", nil)
	if warnItem.Status != "warning" || warnItem.Percent != 84.0 {
		t.Fatalf("expected warning/84.0, got %s/%.1f", warnItem.Status, warnItem.Percent)
	}

	critItem := buildQuotaItem("runs", "AI Runs", 29, 30, "runs", nil)
	if critItem.Status != "critical" {
		t.Fatalf("expected critical, got %s", critItem.Status)
	}

	unlim := buildQuotaItem("monitors", "Active Monitors", 120, -1, "monitors", nil)
	if unlim.Status != "unlimited" || unlim.Percent != 0 {
		t.Fatalf("expected unlimited/0, got %s/%.1f", unlim.Status, unlim.Percent)
	}
}

func TestCalculateBillingCycleAndProjection(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	cycle := calculateBillingCycle(now)
	if cycle.DaysTotal != 30 {
		t.Fatalf("expected 30 days in November, got %d", cycle.DaysTotal)
	}
	if cycle.DaysElapsed != 15 || cycle.DaysRemaining != 15 {
		t.Fatalf("expected 15 elapsed / 15 remaining, got %d / %d", cycle.DaysElapsed, cycle.DaysRemaining)
	}

	proj := projectMonthlyUsage(15, cycle)
	if proj == nil || *proj != 30.0 {
		t.Fatalf("expected projection 30.0, got %v", proj)
	}
}

func TestBuildDateSeries(t *testing.T) {
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 5, 23, 59, 0, 0, time.UTC)
	dates, byDate := buildDateSeries(from, to)
	if len(dates) != 5 || len(byDate) != 5 {
		t.Fatalf("expected 5 days, got %d", len(dates))
	}
	if dates[0] != "2026-11-01" || dates[4] != "2026-11-05" {
		t.Fatalf("unexpected boundaries: %s .. %s", dates[0], dates[4])
	}
}
