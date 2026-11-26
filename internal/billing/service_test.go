package billing

import (
	"context"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	_ "github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBillingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&entities.Team{},
		&entities.TeamEntitlement{},
		&entities.TeamSubscription{},
		&entities.StripeWebhookEvent{},
	))
	return db
}

func TestNormalizeTierAndCatalog(t *testing.T) {
	assert.Equal(t, entities.EntitlementTierFree, entities.NormalizeTier(""))
	assert.Equal(t, entities.EntitlementTierFree, entities.NormalizeTier("free"))
	assert.Equal(t, entities.EntitlementTierPro, entities.NormalizeTier("team"))
	assert.Equal(t, entities.EntitlementTierPro, entities.NormalizeTier("PRO"))
	assert.Equal(t, entities.EntitlementTierEnterprise, entities.NormalizeTier("enterprise"))

	cat := DefaultBillingCatalog()
	assert.Equal(t, "eur", cat.Currency)
	require.Len(t, cat.Plans, 3)
	assert.Equal(t, int64(0), cat.Plans[0].Price)
	assert.Equal(t, int64(20), cat.Plans[1].Price)
	assert.Equal(t, int64(100), cat.Plans[2].Price)
	assert.Len(t, cat.Addons, 4)
}

func TestValidateReturnURL(t *testing.T) {
	assert.Equal(t, "https://my.opsway.eu/settings/billing", ValidateReturnURL("", "https://my.opsway.eu"))
	assert.Equal(t, "https://monitoring.opsway.eu/settings/billing", ValidateReturnURL("https://monitoring.opsway.eu/settings/billing", "https://my.opsway.eu"))
	assert.Equal(t, "https://logs.opsway.eu/billing", ValidateReturnURL("https://logs.opsway.eu/billing", "https://my.opsway.eu"))
	assert.Equal(t, "https://agent-context.opsway.eu/settings/billing", ValidateReturnURL("https://agent-context.opsway.eu/settings/billing", "https://my.opsway.eu"))
	assert.Equal(t, "http://localhost:5173/settings/billing", ValidateReturnURL("http://localhost:5173/settings/billing", "https://my.opsway.eu"))
	// Reject open redirects
	assert.Equal(t, "https://my.opsway.eu/settings/billing", ValidateReturnURL("https://evil.example.com/phish", "https://my.opsway.eu"))
	assert.Equal(t, "https://my.opsway.eu/settings/billing", ValidateReturnURL("javascript:alert(1)", "https://my.opsway.eu"))
}

func TestRecordWebhookEventIdempotency(t *testing.T) {
	db := setupBillingTestDB(t)
	svc := NewService(Config{Domain: "https://my.opsway.eu"}, db)

	dup, err := svc.RecordWebhookEvent(context.Background(), "evt_123", "invoice.paid")
	require.NoError(t, err)
	assert.False(t, dup)

	dup, err = svc.RecordWebhookEvent(context.Background(), "evt_123", "invoice.paid")
	require.NoError(t, err)
	assert.True(t, dup)
}

func TestSyncSubscriptionHybridSuiteAndAddons(t *testing.T) {
	db := setupBillingTestDB(t)
	team := entities.Team{Name: "acme", PaymentPlan: entities.PaymentPlanFree}
	require.NoError(t, db.Create(&team).Error)

	svc := NewService(Config{
		Domain:            "https://my.opsway.eu",
		ProPriceID:        "price_suite_pro",
		EnterprisePriceID: "price_suite_ent",
		AddonPriceIDs: map[string]string{
			"log_search_enterprise": "price_addon_logs_ent",
		},
	}, db)

	periodEnd := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Second)
	sub := &stripe.Subscription{
		ID:                 "sub_001",
		Status:             stripe.SubscriptionStatusActive,
		Customer:           &stripe.Customer{ID: "cus_001"},
		CurrentPeriodStart: time.Now().UTC().Unix(),
		CurrentPeriodEnd:   periodEnd.Unix(),
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{ID: "si_suite", Price: &stripe.Price{ID: "price_suite_pro", LookupKey: "pro"}},
				{ID: "si_logs", Price: &stripe.Price{ID: "price_addon_logs_ent", LookupKey: "addon_log_search_enterprise"}},
			},
		},
	}

	require.NoError(t, svc.SyncSubscriptionFromStripe(context.Background(), team.ID, sub))

	// Base team plan is PRO
	var updatedTeam entities.Team
	require.NoError(t, db.First(&updatedTeam, team.ID).Error)
	assert.Equal(t, entities.PaymentPlanPro, updatedTeam.PaymentPlan)

	// Subscription record persisted
	subRec, err := svc.GetSubscriptionRecord(context.Background(), team.ID)
	require.NoError(t, err)
	require.NotNil(t, subRec)
	assert.Equal(t, "sub_001", subRec.StripeSubscriptionID)
	assert.Equal(t, entities.EntitlementTierPro, subRec.BaseTier)

	// Monitoring is PRO (from Suite), LogSearch is ENTERPRISE (from Add-on)
	monEnt, err := svc.GetEntitlement(context.Background(), team.ID, entities.ProductMonitoring)
	require.NoError(t, err)
	assert.Equal(t, entities.EntitlementTierPro, monEnt.Tier)
	assert.Equal(t, entities.EntitlementSourceSuite, monEnt.Source)
	assert.Equal(t, 50, monEnt.MonthlyQuota)

	logsEnt, err := svc.GetEntitlement(context.Background(), team.ID, entities.ProductLogSearch)
	require.NoError(t, err)
	assert.Equal(t, entities.EntitlementTierEnterprise, logsEnt.Tier)
	assert.Equal(t, entities.EntitlementSourceAddon, logsEnt.Source)
	require.NotNil(t, logsEnt.StripeSubscriptionItemID)
	assert.Equal(t, "si_logs", *logsEnt.StripeSubscriptionItemID)
	assert.Equal(t, -1, logsEnt.MonthlyQuota)

	// Downgrade resets all entitlements to FREE and clears addon subscription item IDs
	require.NoError(t, svc.DowngradeTeamToFree(context.Background(), team.ID))
	logsAfter, err := svc.GetEntitlement(context.Background(), team.ID, entities.ProductLogSearch)
	require.NoError(t, err)
	assert.Equal(t, entities.EntitlementTierFree, logsAfter.Tier)
	assert.Equal(t, entities.EntitlementSourceSuite, logsAfter.Source)
	assert.Nil(t, logsAfter.StripeSubscriptionItemID)
	assert.Equal(t, 1, logsAfter.MonthlyQuota)
}

func TestResetPeriodUsageAndSetProductEnabled(t *testing.T) {
	db := setupBillingTestDB(t)
	team := entities.Team{Name: "beta", PaymentPlan: entities.PaymentPlanFree}
	require.NoError(t, db.Create(&team).Error)

	svc := NewService(Config{}, db)
	ents, err := svc.GetEntitlements(context.Background(), team.ID)
	require.NoError(t, err)
	assert.Len(t, ents, 4)

	// Simulate usage
	require.NoError(t, db.Model(&entities.TeamEntitlement{}).
		Where("team_id = ? AND product = ?", team.ID, entities.ProductManagedAgent).
		Update("current_usage", 3).Error)

	// Reset usage
	nextReset := time.Now().UTC().Add(30 * 24 * time.Hour)
	require.NoError(t, svc.ResetPeriodUsage(context.Background(), team.ID, nextReset))

	agentEnt, err := svc.GetEntitlement(context.Background(), team.ID, entities.ProductManagedAgent)
	require.NoError(t, err)
	assert.Equal(t, 0, agentEnt.CurrentUsage)

	// SetProductEnabled toggles enabled without changing tier
	updated, err := svc.SetProductEnabled(context.Background(), team.ID, entities.ProductPantomath, false)
	require.NoError(t, err)
	assert.False(t, updated.Enabled)
	assert.Equal(t, entities.EntitlementTierFree, updated.Tier)
}
