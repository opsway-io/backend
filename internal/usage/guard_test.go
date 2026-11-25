package usage

import (
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
)

func TestComputeEnforcementTransition_FreeShorterGrace(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

	// 85% -> WARNING
	st, _, _, _ := ComputeEnforcementTransition(entities.PaymentPlanFree, false, nil, "ingestion_gb", 0.85, 1, nil, now)
	if st != entities.QuotaStateWarning {
		t.Fatalf("expected WARNING at 85%%, got %s", st)
	}

	// 105% -> GRACE with 24h expiry
	st, gStart, gEnd, _ := ComputeEnforcementTransition(entities.PaymentPlanFree, false, nil, "ingestion_gb", 1.05, 1, nil, now)
	if st != entities.QuotaStateGrace {
		t.Fatalf("expected GRACE at 105%%, got %s", st)
	}
	if gEnd == nil || gEnd.Sub(*gStart) != 24*time.Hour {
		t.Fatalf("expected 24h Free grace window, got %v", gEnd)
	}

	// 110% on Free -> THROTTLED (+10% cap)
	st, _, _, _ = ComputeEnforcementTransition(entities.PaymentPlanFree, false, nil, "ingestion_gb", 1.10, 1, gStart, now.Add(2*time.Hour))
	if st != entities.QuotaStateThrottled {
		t.Fatalf("expected THROTTLED at 110%% on Free, got %s", st)
	}

	// 105% after 25h on Free -> THROTTLED (24h expired)
	st, _, _, _ = ComputeEnforcementTransition(entities.PaymentPlanFree, false, nil, "ingestion_gb", 1.05, 1, gStart, now.Add(25*time.Hour))
	if st != entities.QuotaStateThrottled {
		t.Fatalf("expected THROTTLED after 24h on Free, got %s", st)
	}
}

func TestComputeEnforcementTransition_PaidGraceAndPayAsYouGo(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

	// Paid without overage at 112% -> still GRACE (under 115% and 72h)
	st, gStart, gEnd, _ := ComputeEnforcementTransition(entities.PaymentPlanTeam, false, nil, "ingestion_gb", 56, 50, nil, now)
	if st != entities.QuotaStateGrace {
		t.Fatalf("expected GRACE at 112%% on Team, got %s", st)
	}
	if gEnd == nil || gEnd.Sub(*gStart) != 72*time.Hour {
		t.Fatalf("expected 72h Paid grace window, got %v", gEnd)
	}

	// Paid WITH pay-as-you-go overage at 160% -> OVERAGE_BILLING
	st, _, _, cents := ComputeEnforcementTransition(entities.PaymentPlanTeam, true, nil, "ingestion_gb", 80, 50, nil, now)
	if st != entities.QuotaStateOverageBilling {
		t.Fatalf("expected OVERAGE_BILLING when AllowOverage=true, got %s", st)
	}
	if cents != 1500 { // 30 GB * 50 cents
		t.Fatalf("expected 1500 cents overage, got %d", cents)
	}

	// Paid WITH pay-as-you-go overage AND spend cap of $10 (1000 cents) -> THROTTLED at $15
	capCents := 1000
	st, _, _, _ = ComputeEnforcementTransition(entities.PaymentPlanTeam, true, &capCents, "ingestion_gb", 80, 50, nil, now)
	if st != entities.QuotaStateThrottled {
		t.Fatalf("expected THROTTLED when spend cap exceeded, got %s", st)
	}
}
