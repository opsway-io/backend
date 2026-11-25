package usage

import (
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReconcilePantomathLifecycle_DowngradeAndUpgrade(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&entities.PantomathProductLifecycle{}, &entities.Team{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	db.Create(&entities.Team{ID: 1, Name: "test-team", PaymentPlan: entities.PaymentPlanFree})

	g := &Guard{db: db}

	// 1. Downgraded to FREE with 2 indexed products -> only 1 allowed, other scheduled for delete in 7 days
	allowed, lc := g.ReconcilePantomathLifecycle(
		t.Context(),
		1,
		entities.PaymentPlanFree,
		false,
		[]string{"log_search", "status_monitoring"},
	)
	if len(allowed) != 1 {
		t.Fatalf("expected 1 allowed product on Free, got %v", allowed)
	}
	if lc["log_search"].Status != entities.PantomathStatusActive {
		t.Fatalf("expected log_search ACTIVE, got %s", lc["log_search"].Status)
	}
	if lc["status_monitoring"].Status != entities.PantomathStatusScheduledDelete || lc["status_monitoring"].DeleteAt == nil {
		t.Fatalf("expected status_monitoring SCHEDULED_DELETE with deleteAt, got %+v", lc["status_monitoring"])
	}

	// 2. Swap active product to status_monitoring
	if err := g.SetActivePantomathProduct(t.Context(), 1, "status_monitoring"); err != nil {
		t.Fatalf("failed to swap active product: %v", err)
	}
	allowed, lc = g.ReconcilePantomathLifecycle(
		t.Context(),
		1,
		entities.PaymentPlanFree,
		false,
		[]string{"log_search", "status_monitoring"},
	)
	if len(allowed) != 1 || allowed[0] != "status_monitoring" {
		t.Fatalf("expected status_monitoring active after swap, got %v", allowed)
	}
	if lc["log_search"].Status != entities.PantomathStatusScheduledDelete {
		t.Fatalf("expected log_search scheduled for delete after swap, got %s", lc["log_search"].Status)
	}

	// 3. Upgrade to TEAM -> both restored to ACTIVE and scheduled deletion cancelled
	allowed, lc = g.ReconcilePantomathLifecycle(
		t.Context(),
		1,
		entities.PaymentPlanTeam,
		false,
		[]string{"log_search", "status_monitoring"},
	)
	if len(allowed) != 2 {
		t.Fatalf("expected 2 allowed products after upgrade to Team, got %v", allowed)
	}
	if lc["log_search"].DeleteAt != nil || lc["log_search"].Status != entities.PantomathStatusActive {
		t.Fatalf("expected scheduled deletion cancelled on upgrade, got %+v", lc["log_search"])
	}
}
