package usage

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const PantomathDowngradeGracePeriod = 7 * 24 * time.Hour

// SyncPlanEntitlements updates all team_entitlements rows (and clears pay-as-you-go on Free)
// whenever a team upgrades or downgrades its Stripe subscription.
func (g *Guard) SyncPlanEntitlements(ctx context.Context, team *entities.Team) error {
	if g == nil || g.db == nil || team == nil {
		return nil
	}
	targetTier := team.MaxEntitlementTier()
	products := []entities.ProductSlug{
		entities.ProductMonitoring,
		entities.ProductLogSearch,
		entities.ProductPantomath,
		entities.ProductManagedAgent,
	}
	for _, prod := range products {
		quota := entities.DefaultMonthlyQuota(prod, targetTier)
		row := entities.TeamEntitlement{
			TeamID:       team.ID,
			Product:      prod,
			Tier:         targetTier,
			Enabled:      true,
			MonthlyQuota: quota,
		}
		if err := g.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "team_id"}, {Name: "product"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"tier":          targetTier,
				"monthly_quota": quota,
				"updated_at":    time.Now().UTC(),
			}),
		}).Create(&row).Error; err != nil && g.logger != nil {
			g.logger.WithError(err).Warn("failed to sync team entitlement on plan change")
		}
	}

	if team.PaymentPlan == entities.PaymentPlanFree && team.AllowOverage {
		_ = g.db.WithContext(ctx).Model(&entities.Team{}).Where("id = ?", team.ID).Updates(map[string]interface{}{
			"allow_overage":           false,
			"overage_spend_cap_cents": nil,
		}).Error
	}

	_, _ = g.EvaluateTeam(ctx, team.ID)
	return nil
}

// isPlatformAdminTeam reports whether any member of teamID is a platform owner.
func isPlatformAdminTeam(ctx context.Context, db *gorm.DB, teamID uint) bool {
	if db == nil || teamID == 0 {
		return false
	}
	var count int64
	err := db.WithContext(ctx).
		Table("team_users").
		Joins("JOIN users ON users.id = team_users.user_id").
		Where("team_users.team_id = ? AND users.is_platform_owner = ?", teamID, true).
		Limit(1).
		Count(&count).Error
	return err == nil && count > 0
}

// ReconcileTeamStripePlan verifies a team's active Stripe subscription and ensures
// team.PaymentPlan and all product entitlements match paid Stripe state.
// Platform-admin teams are always granted Enterprise across all products.
func (g *Guard) ReconcileTeamStripePlan(ctx context.Context, team *entities.Team) {
	if g == nil || g.db == nil || team == nil {
		return
	}

	if isPlatformAdminTeam(ctx, g.db, team.ID) {
		if team.PaymentPlan != entities.PaymentPlanEnterprise {
			team.PaymentPlan = entities.PaymentPlanEnterprise
			_ = g.db.WithContext(ctx).Model(&entities.Team{}).Where("id = ?", team.ID).Update("payment_plan", entities.PaymentPlanEnterprise).Error
			_ = g.SyncPlanEntitlements(ctx, team)
		}
		return
	}

	if g.billingSvc == nil || !g.billingSvc.IsConfigured() {
		return
	}

	resolvedPlan := entities.PaymentPlanFree
	if team.StripeCustomerID != nil && *team.StripeCustomerID != "" {
		iter := g.billingSvc.GetCustomerSubscription(*team.StripeCustomerID)
		if iter == nil {
			return
		}
		for iter.Next() {
			if p := g.billingSvc.ResolvePlanFromSubscription(iter.Subscription()); p != entities.PaymentPlanFree {
				resolvedPlan = p
				break
			}
		}
		if err := iter.Err(); err != nil {
			if g.logger != nil {
				g.logger.WithError(err).WithField("teamId", team.ID).Warn("stripe reconciliation: failed to query subscriptions")
			}
			return
		}
	}

	if team.PaymentPlan != resolvedPlan {
		if g.logger != nil {
			g.logger.WithFields(map[string]interface{}{
				"teamId": team.ID,
				"from":   team.PaymentPlan,
				"to":     resolvedPlan,
			}).Info("stripe reconciliation: syncing team plan to match paid Stripe subscription")
		}
		team.PaymentPlan = resolvedPlan
		_ = g.db.WithContext(ctx).Model(&entities.Team{}).Where("id = ?", team.ID).Update("payment_plan", resolvedPlan).Error
		_ = g.SyncPlanEntitlements(ctx, team)
	}
}

// ReconcilePantomathLifecycle enforces the plan's max indexed products limit:
//   - Only up to `maxProducts` remain ACTIVE and searchable.
//   - Excess products are marked SCHEDULED_DELETE with deleteAt = now + 7 days.
//   - Upgrading (or paying for overages) cancels scheduled deletions and restores ACTIVE.
//   - Any SCHEDULED_DELETE product past its 7-day deleteAt is purged from Pantomath Milvus.
func (g *Guard) ReconcilePantomathLifecycle(
	ctx context.Context,
	teamID uint,
	plan entities.PaymentPlan,
	allowOverage bool,
	discoveredProducts []string,
) (allowedProducts []string, lifecycleByProduct map[string]entities.PantomathProductLifecycle) {
	lifecycleByProduct = make(map[string]entities.PantomathProductLifecycle)
	if g == nil || g.db == nil {
		return discoveredProducts, lifecycleByProduct
	}

	now := time.Now().UTC()
	var existing []entities.PantomathProductLifecycle
	g.db.WithContext(ctx).Where("team_id = ?", teamID).Find(&existing)
	existingMap := make(map[string]entities.PantomathProductLifecycle, len(existing))
	for _, row := range existing {
		existingMap[row.Product] = row
	}

	// Merge discovered + existing non-deleted products
	prodSet := make(map[string]struct{})
	for _, p := range discoveredProducts {
		if p != "" {
			prodSet[p] = struct{}{}
		}
	}
	for _, row := range existing {
		if row.Status != entities.PantomathStatusDeleted {
			prodSet[row.Product] = struct{}{}
		}
	}
	allProducts := make([]string, 0, len(prodSet))
	for p := range prodSet {
		allProducts = append(allProducts, p)
	}
	sort.Strings(allProducts)

	if len(allProducts) == 0 {
		return nil, lifecycleByProduct
	}

	team := entities.Team{PaymentPlan: plan}
	maxProducts := entities.DefaultMonthlyQuota(entities.ProductPantomath, team.MaxEntitlementTier())
	isPaidOverage := allowOverage && (plan == entities.PaymentPlanTeam || plan == entities.PaymentPlanEnterprise)

	// Within limit (or Enterprise / paid overage): restore any SCHEDULED_DELETE products back to ACTIVE.
	if maxProducts < 0 || isPaidOverage || len(allProducts) <= maxProducts {
		for _, p := range allProducts {
			row := existingMap[p]
			row.TeamID = teamID
			row.Product = p
			row.Status = entities.PantomathStatusActive
			row.DeleteAt = nil
			g.upsertLifecycle(ctx, &row)
			lifecycleByProduct[p] = row
			allowedProducts = append(allowedProducts, p)
		}
		return allowedProducts, lifecycleByProduct
	}

	// Over limit (e.g., 2 indexed products on Free where maxProducts == 1):
	// Prioritize products that are already ACTIVE, then fill remaining slots deterministically.
	selected := make(map[string]bool)
	for _, p := range allProducts {
		if len(selected) >= maxProducts {
			break
		}
		if row, ok := existingMap[p]; ok && row.Status == entities.PantomathStatusActive {
			selected[p] = true
		}
	}
	for _, p := range allProducts {
		if len(selected) >= maxProducts {
			break
		}
		if !selected[p] {
			selected[p] = true
		}
	}

	for _, p := range allProducts {
		row := existingMap[p]
		row.TeamID = teamID
		row.Product = p
		if selected[p] {
			row.Status = entities.PantomathStatusActive
			row.DeleteAt = nil
			g.upsertLifecycle(ctx, &row)
			lifecycleByProduct[p] = row
			allowedProducts = append(allowedProducts, p)
			continue
		}

		// Excess product: schedule deletion in 7 days (preserving original deleteAt if already scheduled)
		row.Status = entities.PantomathStatusScheduledDelete
		if row.DeleteAt == nil {
			delAt := now.Add(PantomathDowngradeGracePeriod)
			row.DeleteAt = &delAt
		}

		if row.DeleteAt != nil && !now.Before(*row.DeleteAt) {
			if g.purgePantomathProduct(ctx, teamID, p) {
				row.Status = entities.PantomathStatusDeleted
			}
		}

		g.upsertLifecycle(ctx, &row)
		lifecycleByProduct[p] = row
	}

	return allowedProducts, lifecycleByProduct
}

// CanAddPantomathProduct checks whether a team can create (or add repos to) a Pantomath product
// under its active plan limits. Existing non-deleted products are always allowed.
func (g *Guard) CanAddPantomathProduct(ctx context.Context, teamID uint, product string) (allowed bool, currentCount int, maxProducts int) {
	if g == nil || g.db == nil {
		return true, 0, -1
	}
	product = strings.TrimSpace(product)
	var team entities.Team
	if err := g.db.WithContext(ctx).Select("id, payment_plan, allow_overage").First(&team, teamID).Error; err != nil {
		return true, 0, -1
	}
	maxProducts = entities.DefaultMonthlyQuota(entities.ProductPantomath, team.MaxEntitlementTier())
	isPaidOverage := team.AllowOverage && (team.PaymentPlan == entities.PaymentPlanTeam || team.PaymentPlan == entities.PaymentPlanEnterprise)

	var rows []entities.PantomathProductLifecycle
	g.db.WithContext(ctx).Where("team_id = ? AND status != ?", teamID, entities.PantomathStatusDeleted).Find(&rows)
	for _, r := range rows {
		if r.Product == product {
			return true, len(rows), maxProducts
		}
	}
	currentCount = len(rows)
	if maxProducts < 0 || isPaidOverage || currentCount < maxProducts {
		return true, currentCount, maxProducts
	}
	return false, currentCount, maxProducts
}

// RegisterPantomathProduct ensures a newly created product is immediately recorded as ACTIVE
// in pantomath_product_lifecycles so quota-state allows querying it right away.
func (g *Guard) RegisterPantomathProduct(ctx context.Context, teamID uint, product string) error {
	if g == nil || g.db == nil {
		return nil
	}
	product = strings.TrimSpace(product)
	if product == "" {
		return fmt.Errorf("product is required")
	}
	row := entities.PantomathProductLifecycle{
		TeamID:   teamID,
		Product:  product,
		Status:   entities.PantomathStatusActive,
		DeleteAt: nil,
	}
	g.upsertLifecycle(ctx, &row)
	return nil
}

// SetActivePantomathProduct lets a team choose which product(s) stay ACTIVE when downgraded.
func (g *Guard) SetActivePantomathProduct(ctx context.Context, teamID uint, product string) error {
	product = strings.TrimSpace(product)
	if product == "" {
		return fmt.Errorf("product is required")
	}
	var team entities.Team
	if err := g.db.WithContext(ctx).First(&team, teamID).Error; err != nil {
		return err
	}
	maxProducts := entities.DefaultMonthlyQuota(entities.ProductPantomath, team.MaxEntitlementTier())

	var rows []entities.PantomathProductLifecycle
	g.db.WithContext(ctx).Where("team_id = ? AND status != ?", teamID, entities.PantomathStatusDeleted).Find(&rows)

	now := time.Now().UTC()
	delAt := now.Add(PantomathDowngradeGracePeriod)

	// Mark target product ACTIVE, and if maxProducts == 1, demote any other ACTIVE product to SCHEDULED_DELETE.
	found := false
	for i := range rows {
		if rows[i].Product == product {
			rows[i].Status = entities.PantomathStatusActive
			rows[i].DeleteAt = nil
			found = true
		} else if maxProducts == 1 && rows[i].Status == entities.PantomathStatusActive {
			rows[i].Status = entities.PantomathStatusScheduledDelete
			if rows[i].DeleteAt == nil {
				rows[i].DeleteAt = &delAt
			}
		}
		g.upsertLifecycle(ctx, &rows[i])
	}
	if !found {
		row := entities.PantomathProductLifecycle{
			TeamID:  teamID,
			Product: product,
			Status:  entities.PantomathStatusActive,
		}
		g.upsertLifecycle(ctx, &row)
	}
	return nil
}

func (g *Guard) GetPantomathAllowedProducts(ctx context.Context, teamID uint) (allowed []string, scheduledDelete []string) {
	var team entities.Team
	if err := g.db.WithContext(ctx).Select("id, payment_plan, allow_overage").First(&team, teamID).Error; err != nil {
		return nil, nil
	}
	var rows []entities.PantomathProductLifecycle
	g.db.WithContext(ctx).Where("team_id = ? AND status != ?", teamID, entities.PantomathStatusDeleted).Find(&rows)
	if len(rows) == 0 {
		return nil, nil
	}
	var discovered []string
	for _, r := range rows {
		discovered = append(discovered, r.Product)
	}
	allowed, lc := g.ReconcilePantomathLifecycle(ctx, teamID, team.PaymentPlan, team.AllowOverage, discovered)
	for p, r := range lc {
		if r.Status == entities.PantomathStatusScheduledDelete {
			scheduledDelete = append(scheduledDelete, p)
		}
	}
	return allowed, scheduledDelete
}

func (g *Guard) upsertLifecycle(ctx context.Context, row *entities.PantomathProductLifecycle) {
	_ = g.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "team_id"}, {Name: "product"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"status":     row.Status,
			"delete_at":  row.DeleteAt,
			"updated_at": time.Now().UTC(),
		}),
	}).Create(row).Error
}

func (g *Guard) purgePantomathProduct(ctx context.Context, teamID uint, product string) bool {
	pantoURL := ""
	if svc, ok := g.usageSvc.(*serviceImpl); ok {
		pantoURL = svc.pantomathURL
	}
	if pantoURL == "" {
		return false
	}
	u := fmt.Sprintf("%s/v1/products/%s/purge?tenant=team_%d", pantoURL, product, teamID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return false
	}
	resp, err := g.httpClient.Do(req)
	if err != nil {
		if g.logger != nil {
			g.logger.WithError(err).WithField("product", product).Warn("failed to purge expired Pantomath product")
		}
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
