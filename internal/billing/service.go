package billing

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/billingportal/configuration"
	portalsession "github.com/stripe/stripe-go/v81/billingportal/session"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customersession"
	"github.com/stripe/stripe-go/v81/price"
	"github.com/stripe/stripe-go/v81/product"
	"github.com/stripe/stripe-go/v81/subscription"
	"github.com/stripe/stripe-go/v81/subscriptionitem"
	"github.com/stripe/stripe-go/v81/webhook"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Config struct {
	PublishableKey    string            `mapstructure:"publishable_key"`
	SecretKey         string            `mapstructure:"secret_key"`
	WebhookSecret     string            `mapstructure:"webhook_secret"`
	Domain            string            `mapstructure:"domain"`
	Currency          string            `mapstructure:"currency"`
	ProPriceID        string            `mapstructure:"pro_price_id"`
	TeamPriceID       string            `mapstructure:"team_price_id"`
	EnterprisePriceID string            `mapstructure:"enterprise_price_id"`
	AddonPriceIDs     map[string]string `mapstructure:"addon_price_ids"`
}

func (c Config) EffectiveProPriceID() string {
	if strings.TrimSpace(c.ProPriceID) != "" {
		return strings.TrimSpace(c.ProPriceID)
	}
	return strings.TrimSpace(c.TeamPriceID)
}

func (c Config) EffectiveDomain() string {
	if strings.TrimSpace(c.Domain) != "" {
		return strings.TrimRight(strings.TrimSpace(c.Domain), "/")
	}
	return "https://my.opsway.eu"
}

type Service interface {
	IsConfigured() bool
	Domain() string
	GetCatalog() BillingCatalog
	ResolvePlanFromSubscription(sub *stripe.Subscription) entities.PaymentPlan
	PostConfig() StripeConfig
	CreateCheckoutSession(team *entities.Team, priceID string) (*stripe.CheckoutSession, error)
	CreateCheckoutSessionWithReturn(team *entities.Team, priceID string, returnURL string, metadata map[string]string) (*stripe.CheckoutSession, error)
	UpdateSubscription(team *entities.Team, priceID string) (*stripe.Subscription, error)
	CancelSubscription(team *entities.Team) (*stripe.Subscription, error)
	ResumeSubscription(team *entities.Team) (*stripe.Subscription, error)
	UpsertProductAddon(ctx context.Context, team *entities.Team, prod entities.ProductSlug, tier entities.EntitlementTier, returnURL string) (checkoutURL string, err error)
	GetCheckoutSession(sessionID string) (*stripe.CheckoutSession, error)
	GetLineItems(sessionID string) *session.LineItemIter
	GetPrices(priceLookupKeys []string) ([]*stripe.Price, error)
	GetCustomerSubscription(customerID string) *subscription.Iter
	GetSubscription(subID string) (*stripe.Subscription, error)
	GetProduct(productID string) (*stripe.Product, error)
	GetProducts() *product.Iter
	CreateCustomerPortal(team *entities.Team) (*stripe.BillingPortalSession, error)
	CreateCustomerPortalWithReturn(team *entities.Team, returnURL string) (*stripe.BillingPortalSession, error)
	ConstructEvent(payload []byte, header string) (stripe.Event, error)
	GetCustomerSession(team *entities.Team) (*stripe.CustomerSession, error)
	GetBillingPortal(team *entities.Team) (*stripe.BillingPortalConfiguration, error)

	// DB-backed Entitlements, Subscription State & Webhook Idempotency
	RecordWebhookEvent(ctx context.Context, eventID, eventType string) (alreadyProcessed bool, err error)
	GetSubscriptionRecord(ctx context.Context, teamID uint) (*entities.TeamSubscription, error)
	GetEntitlements(ctx context.Context, teamID uint) ([]entities.TeamEntitlement, error)
	GetEntitlement(ctx context.Context, teamID uint, prod entities.ProductSlug) (*entities.TeamEntitlement, error)
	SetProductEnabled(ctx context.Context, teamID uint, prod entities.ProductSlug, enabled bool) (*entities.TeamEntitlement, error)
	SyncSubscriptionFromStripe(ctx context.Context, teamID uint, sub *stripe.Subscription) error
	DowngradeTeamToFree(ctx context.Context, teamID uint) error
	ResetPeriodUsage(ctx context.Context, teamID uint, nextResetAt time.Time) error
	ResetExpiredFreeUsage(ctx context.Context) error
}

type ServiceImpl struct {
	Config Config
	db     *gorm.DB
}

func NewService(conf Config, db *gorm.DB) Service {
	stripe.Key = conf.SecretKey
	return &ServiceImpl{Config: conf, db: db}
}

type StripeConfig struct {
	PublishableKey  string `json:"publishableKey"`
	ProPrice        string `json:"proPrice"`
	TeamPrice       string `json:"teamPrice"`
	EnterprisePrice string `json:"enterprisePrice"`
}

func (s *ServiceImpl) IsConfigured() bool {
	return strings.TrimSpace(s.Config.SecretKey) != ""
}

func (s *ServiceImpl) Domain() string {
	return s.Config.EffectiveDomain()
}

func (s *ServiceImpl) GetCatalog() BillingCatalog {
	return DefaultBillingCatalog()
}

// parseAddonLookupKey parses lookup keys of the form "addon_<product>_<tier>"
// (e.g. "addon_log_search_pro") into (product, tier, true).
func parseAddonLookupKey(lookupKey string) (entities.ProductSlug, entities.EntitlementTier, bool) {
	k := strings.ToLower(strings.TrimSpace(lookupKey))
	if !strings.HasPrefix(k, "addon_") {
		return "", "", false
	}
	rest := strings.TrimPrefix(k, "addon_")
	for _, prod := range entities.AllProducts() {
		prefix := string(prod) + "_"
		if strings.HasPrefix(rest, prefix) {
			tierStr := strings.TrimPrefix(rest, prefix)
			tier := entities.NormalizeTier(tierStr)
			if tier != entities.EntitlementTierFree {
				return prod, tier, true
			}
		}
	}
	return "", "", false
}

// resolvePriceItem inspects a Stripe subscription item and determines whether it
// represents a base Suite plan or a modular Product Add-on.
func (s *ServiceImpl) resolvePriceItem(item *stripe.SubscriptionItem) (isSuite bool, suiteTier entities.EntitlementTier, addonProduct entities.ProductSlug, addonTier entities.EntitlementTier) {
	if item == nil || item.Price == nil {
		return false, entities.EntitlementTierFree, "", ""
	}
	p := item.Price

	// 1. Check explicit configured AddonPriceIDs map first
	for key, cfgPriceID := range s.Config.AddonPriceIDs {
		if cfgPriceID != "" && p.ID == cfgPriceID {
			if prod, tier, ok := parseAddonLookupKey("addon_" + key); ok {
				return false, "", prod, tier
			}
		}
	}

	// 2. Check addon lookup key or metadata
	if prod, tier, ok := parseAddonLookupKey(p.LookupKey); ok {
		return false, "", prod, tier
	}
	if p.Metadata != nil {
		if rawProd := entities.ProductSlug(strings.ToLower(strings.TrimSpace(p.Metadata["product"]))); entities.IsValidProduct(rawProd) {
			tier := entities.NormalizeTier(p.Metadata["tier"])
			if tier != entities.EntitlementTierFree {
				return false, "", rawProd, tier
			}
		}
	}

	// 3. Base Suite price IDs
	if s.Config.EnterprisePriceID != "" && p.ID == s.Config.EnterprisePriceID {
		return true, entities.EntitlementTierEnterprise, "", ""
	}
	if proID := s.Config.EffectiveProPriceID(); proID != "" && p.ID == proID {
		return true, entities.EntitlementTierPro, "", ""
	}

	// 4. Base Suite lookup keys (including legacy "team")
	switch strings.ToUpper(strings.TrimSpace(p.LookupKey)) {
	case "ENTERPRISE", "SUITE_ENTERPRISE":
		return true, entities.EntitlementTierEnterprise, "", ""
	case "PRO", "TEAM", "SUITE_PRO":
		return true, entities.EntitlementTierPro, "", ""
	default:
		return false, entities.EntitlementTierFree, "", ""
	}
}

// ResolvePlanFromSubscription maps a Stripe subscription to a verified base PaymentPlan.
// Only active or trialing subscriptions grant a paid plan; canceled, unpaid, or
// incomplete_expired subscriptions revert to FREE.
func (s *ServiceImpl) ResolvePlanFromSubscription(sub *stripe.Subscription) entities.PaymentPlan {
	if sub == nil {
		return entities.PaymentPlanFree
	}
	if sub.Status != stripe.SubscriptionStatusActive && sub.Status != stripe.SubscriptionStatusTrialing {
		return entities.PaymentPlanFree
	}
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return entities.PaymentPlanFree
	}

	bestTier := entities.EntitlementTierFree
	for _, item := range sub.Items.Data {
		if isSuite, tier, _, _ := s.resolvePriceItem(item); isSuite {
			if entities.TierRank(tier) > entities.TierRank(bestTier) {
				bestTier = tier
			}
		}
	}

	switch bestTier {
	case entities.EntitlementTierEnterprise:
		return entities.PaymentPlanEnterprise
	case entities.EntitlementTierPro:
		return entities.PaymentPlanPro
	default:
		return entities.PaymentPlanFree
	}
}

func (s *ServiceImpl) PostConfig() StripeConfig {
	proPrice := s.Config.EffectiveProPriceID()
	return StripeConfig{
		PublishableKey:  s.Config.PublishableKey,
		ProPrice:        proPrice,
		TeamPrice:       proPrice,
		EnterprisePrice: s.Config.EnterprisePriceID,
	}
}

func (s *ServiceImpl) CreateCheckoutSession(team *entities.Team, priceID string) (*stripe.CheckoutSession, error) {
	return s.CreateCheckoutSessionWithReturn(team, priceID, "", nil)
}

func (s *ServiceImpl) CreateCheckoutSessionWithReturn(team *entities.Team, priceID string, returnURL string, metadata map[string]string) (*stripe.CheckoutSession, error) {
	if team == nil {
		return nil, errors.New("team is required")
	}
	if strings.TrimSpace(priceID) == "" {
		return nil, errors.New("price ID is required")
	}

	targetURL := ValidateReturnURL(returnURL, s.Domain())
	params := &stripe.CheckoutSessionParams{
		SuccessURL:        stripe.String(targetURL),
		CancelURL:         stripe.String(targetURL),
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		ClientReferenceID: stripe.String(strconv.FormatUint(uint64(team.ID), 10)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	if len(metadata) > 0 {
		params.Metadata = metadata
		params.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: metadata,
		}
	}

	if team.StripeCustomerID != nil && *team.StripeCustomerID != "" {
		params.Customer = stripe.String(*team.StripeCustomerID)
	}

	return session.New(params)
}

func (s *ServiceImpl) UpdateSubscription(team *entities.Team, priceID string) (*stripe.Subscription, error) {
	if team == nil || team.StripeCustomerID == nil || *team.StripeCustomerID == "" {
		return nil, errors.New("team has no Stripe customer ID")
	}

	sub := s.GetCustomerSubscription(*team.StripeCustomerID)
	if !sub.Next() {
		return nil, errors.New("no active subscription found for customer")
	}
	teamSubscription := sub.Subscription()

	if teamSubscription.Items == nil || len(teamSubscription.Items.Data) == 0 {
		return nil, errors.New("subscription has no items")
	}

	// Find the base suite line item if present, otherwise fall back to item 0.
	targetItemID := teamSubscription.Items.Data[0].ID
	for _, item := range teamSubscription.Items.Data {
		if isSuite, _, _, _ := s.resolvePriceItem(item); isSuite {
			targetItemID = item.ID
			break
		}
	}

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
		PaymentBehavior:   stripe.String("error_if_incomplete"),
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(targetItemID),
				Price: stripe.String(priceID),
			},
		},
	}
	result, err := subscription.Update(teamSubscription.ID, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update subscription")
	}

	return result, nil
}

// CancelSubscription schedules cancellation at the end of the current billing period
// so paid teams retain access for the period they already paid for.
func (s *ServiceImpl) CancelSubscription(team *entities.Team) (*stripe.Subscription, error) {
	if team == nil || team.StripeCustomerID == nil || *team.StripeCustomerID == "" {
		return nil, errors.New("team has no Stripe customer ID")
	}

	sub := s.GetCustomerSubscription(*team.StripeCustomerID)
	if !sub.Next() {
		return nil, errors.New("no active subscription found for customer")
	}
	teamSubscription := sub.Subscription()

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	result, err := subscription.Update(teamSubscription.ID, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to schedule subscription cancellation")
	}

	if s.db != nil {
		_ = s.db.Model(&entities.TeamSubscription{}).
			Where("team_id = ?", team.ID).
			Update("cancel_at_period_end", true).Error
	}

	return result, nil
}

// ResumeSubscription clears cancel_at_period_end on an active subscription.
func (s *ServiceImpl) ResumeSubscription(team *entities.Team) (*stripe.Subscription, error) {
	if team == nil || team.StripeCustomerID == nil || *team.StripeCustomerID == "" {
		return nil, errors.New("team has no Stripe customer ID")
	}

	sub := s.GetCustomerSubscription(*team.StripeCustomerID)
	if !sub.Next() {
		return nil, errors.New("no active subscription found for customer")
	}
	teamSubscription := sub.Subscription()

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}
	result, err := subscription.Update(teamSubscription.ID, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resume subscription")
	}

	if s.db != nil {
		_ = s.db.Model(&entities.TeamSubscription{}).
			Where("team_id = ?", team.ID).
			Update("cancel_at_period_end", false).Error
	}

	return result, nil
}

func (s *ServiceImpl) resolveAddonPriceID(prod entities.ProductSlug, tier entities.EntitlementTier) (string, error) {
	key := fmt.Sprintf("%s_%s", strings.ToLower(string(prod)), strings.ToLower(string(tier)))
	if id := strings.TrimSpace(s.Config.AddonPriceIDs[key]); id != "" {
		return id, nil
	}
	lookupKey := "addon_" + key
	prices, err := s.GetPrices([]string{lookupKey})
	if err == nil && len(prices) > 0 {
		return prices[0].ID, nil
	}
	return "", fmt.Errorf("stripe price not configured for add-on %s", lookupKey)
}

// UpsertProductAddon adds, upgrades, or removes a modular product add-on line item
// on the team's Stripe subscription (or starts a Checkout Session if the team has none).
func (s *ServiceImpl) UpsertProductAddon(
	ctx context.Context,
	team *entities.Team,
	prod entities.ProductSlug,
	tier entities.EntitlementTier,
	returnURL string,
) (string, error) {
	if !entities.IsValidProduct(prod) {
		return "", fmt.Errorf("invalid product: %s", prod)
	}
	normalizedTier := entities.NormalizeTier(string(tier))

	// Removing an add-on (downgrade add-on to FREE / base suite tier)
	if normalizedTier == entities.EntitlementTierFree {
		ent, err := s.GetEntitlement(ctx, team.ID, prod)
		if err != nil {
			return "", err
		}
		if ent.StripeSubscriptionItemID != nil && *ent.StripeSubscriptionItemID != "" {
			if _, err := subscriptionitem.Del(*ent.StripeSubscriptionItemID, nil); err != nil {
				return "", errors.Wrap(err, "failed to remove addon subscription item")
			}
		}
		return "", nil
	}

	priceID, err := s.resolveAddonPriceID(prod, normalizedTier)
	if err != nil {
		return "", err
	}

	// If the team already has an active Stripe subscription, mutate its items directly.
	if team.StripeCustomerID != nil && *team.StripeCustomerID != "" {
		iter := s.GetCustomerSubscription(*team.StripeCustomerID)
		if iter.Next() {
			activeSub := iter.Subscription()
			if activeSub.Status == stripe.SubscriptionStatusActive || activeSub.Status == stripe.SubscriptionStatusTrialing {
				ent, _ := s.GetEntitlement(ctx, team.ID, prod)
				if ent != nil && ent.StripeSubscriptionItemID != nil && *ent.StripeSubscriptionItemID != "" {
					_, err := subscriptionitem.Update(*ent.StripeSubscriptionItemID, &stripe.SubscriptionItemParams{
						Price: stripe.String(priceID),
					})
					if err != nil {
						return "", errors.Wrap(err, "failed to update addon subscription item")
					}
				} else {
					_, err := subscriptionitem.New(&stripe.SubscriptionItemParams{
						Subscription: stripe.String(activeSub.ID),
						Price:        stripe.String(priceID),
						Quantity:     stripe.Int64(1),
					})
					if err != nil {
						return "", errors.Wrap(err, "failed to add addon subscription item")
					}
				}
				// Re-fetch and reconcile immediately
				if refreshed, err := s.GetSubscription(activeSub.ID); err == nil {
					_ = s.SyncSubscriptionFromStripe(ctx, team.ID, refreshed)
				}
				return "", nil
			}
		}
	}

	// Otherwise launch Stripe Checkout for the standalone product add-on.
	cs, err := s.CreateCheckoutSessionWithReturn(team, priceID, returnURL, map[string]string{
		"team_id": strconv.FormatUint(uint64(team.ID), 10),
		"product": string(prod),
		"tier":    string(normalizedTier),
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to create addon checkout session")
	}
	return cs.URL, nil
}

func (s *ServiceImpl) GetCheckoutSession(sessionID string) (*stripe.CheckoutSession, error) {
	return session.Get(sessionID, &stripe.CheckoutSessionParams{})
}

func (s *ServiceImpl) GetLineItems(sessionID string) *session.LineItemIter {
	params := &stripe.CheckoutSessionListLineItemsParams{
		Session: stripe.String(sessionID),
	}
	return session.ListLineItems(params)
}

func (s *ServiceImpl) GetPrices(priceLookupKeys []string) ([]*stripe.Price, error) {
	params := &stripe.PriceListParams{
		LookupKeys: stripe.StringSlice(priceLookupKeys),
	}
	i := price.List(params)

	prices := make([]*stripe.Price, 0)
	for i.Next() {
		p := i.Price()
		prices = append(prices, p)
	}
	if err := i.Err(); err != nil {
		return nil, err
	}

	return prices, nil
}

func (s *ServiceImpl) GetCustomerSubscription(customerID string) *subscription.Iter {
	params := &stripe.SubscriptionListParams{Customer: stripe.String(customerID)}
	return subscription.List(params)
}

func (s *ServiceImpl) GetSubscription(subID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{}
	return subscription.Get(subID, params)
}

func (s *ServiceImpl) GetProduct(productID string) (*stripe.Product, error) {
	params := &stripe.ProductParams{}
	return product.Get(productID, params)
}

func (s *ServiceImpl) GetProducts() *product.Iter {
	params := &stripe.ProductListParams{}
	return product.List(params)
}

func (s *ServiceImpl) CreateCustomerPortal(team *entities.Team) (*stripe.BillingPortalSession, error) {
	return s.CreateCustomerPortalWithReturn(team, "")
}

func (s *ServiceImpl) CreateCustomerPortalWithReturn(team *entities.Team, returnURL string) (*stripe.BillingPortalSession, error) {
	if team == nil || team.StripeCustomerID == nil || *team.StripeCustomerID == "" {
		return nil, errors.New("team has no Stripe customer ID")
	}
	targetURL := ValidateReturnURL(returnURL, s.Domain())
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(*team.StripeCustomerID),
		ReturnURL: stripe.String(targetURL),
	}
	return portalsession.New(params)
}

func (s *ServiceImpl) ConstructEvent(payload []byte, header string) (stripe.Event, error) {
	return webhook.ConstructEventWithOptions(payload, header, s.Config.WebhookSecret, webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
}

func (s *ServiceImpl) GetCustomerSession(team *entities.Team) (*stripe.CustomerSession, error) {
	if team == nil || team.StripeCustomerID == nil || *team.StripeCustomerID == "" {
		return nil, errors.New("team has no Stripe customer ID")
	}
	params := &stripe.CustomerSessionParams{
		Customer: stripe.String(*team.StripeCustomerID),
		Components: &stripe.CustomerSessionComponentsParams{
			PricingTable: &stripe.CustomerSessionComponentsPricingTableParams{
				Enabled: stripe.Bool(true),
			},
		},
	}
	return customersession.New(params)
}

func (s *ServiceImpl) GetBillingPortal(team *entities.Team) (*stripe.BillingPortalConfiguration, error) {
	params := &stripe.BillingPortalConfigurationParams{
		Features: &stripe.BillingPortalConfigurationFeaturesParams{
			InvoiceHistory: &stripe.BillingPortalConfigurationFeaturesInvoiceHistoryParams{
				Enabled: stripe.Bool(true),
			},
		},
	}
	return configuration.New(params)
}

// --- DB-backed Entitlements, Subscription State & Webhook Idempotency ---

func nextMonthReset(now time.Time) time.Time {
	utc := now.UTC()
	return time.Date(utc.Year(), utc.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}

func (s *ServiceImpl) RecordWebhookEvent(ctx context.Context, eventID, eventType string) (bool, error) {
	if s.db == nil || strings.TrimSpace(eventID) == "" {
		return false, nil
	}
	rec := entities.StripeWebhookEvent{
		EventID:     eventID,
		EventType:   eventType,
		ProcessedAt: time.Now().UTC(),
	}
	res := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "event_id"}}, DoNothing: true}).
		Create(&rec)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 0, nil
}

func (s *ServiceImpl) GetSubscriptionRecord(ctx context.Context, teamID uint) (*entities.TeamSubscription, error) {
	if s.db == nil {
		return nil, nil
	}
	var rec entities.TeamSubscription
	err := s.db.WithContext(ctx).Where("team_id = ?", teamID).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *ServiceImpl) GetEntitlements(ctx context.Context, teamID uint) ([]entities.TeamEntitlement, error) {
	if s.db == nil {
		return nil, nil
	}
	var entitlements []entities.TeamEntitlement
	if err := s.db.WithContext(ctx).Where("team_id = ?", teamID).Order("id asc").Find(&entitlements).Error; err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	resetAt := nextMonthReset(now)

	existingByProd := make(map[entities.ProductSlug]entities.TeamEntitlement, len(entitlements))
	for i := range entitlements {
		ent := &entitlements[i]
		// Lazy reset if period elapsed
		if ent.UsageResetAt != nil && !now.Before(*ent.UsageResetAt) {
			next := nextMonthReset(now)
			ent.CurrentUsage = 0
			ent.UsageResetAt = &next
			_ = s.db.WithContext(ctx).Model(ent).Updates(map[string]any{
				"current_usage":  0,
				"usage_reset_at": next,
			}).Error
		}
		existingByProd[ent.Product] = *ent
	}

	// Ensure all 4 products exist
	if len(existingByProd) < len(entities.AllProducts()) {
		var team entities.Team
		baseTier := entities.EntitlementTierFree
		if err := s.db.WithContext(ctx).Select("payment_plan").First(&team, teamID).Error; err == nil {
			baseTier = entities.NormalizeTier(string(team.PaymentPlan))
		}
		for _, prod := range entities.AllProducts() {
			if _, ok := existingByProd[prod]; ok {
				continue
			}
			ent := entities.TeamEntitlement{
				TeamID:       teamID,
				Product:      prod,
				Tier:         baseTier,
				Source:       entities.EntitlementSourceSuite,
				Enabled:      true,
				MonthlyQuota: entities.DefaultMonthlyQuota(prod, baseTier),
				CurrentUsage: 0,
				UsageResetAt: &resetAt,
			}
			_ = s.db.WithContext(ctx).
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&ent).Error
			entitlements = append(entitlements, ent)
		}
	}

	return entitlements, nil
}

func (s *ServiceImpl) GetEntitlement(ctx context.Context, teamID uint, prod entities.ProductSlug) (*entities.TeamEntitlement, error) {
	ents, err := s.GetEntitlements(ctx, teamID)
	if err != nil {
		return nil, err
	}
	for i := range ents {
		if ents[i].Product == prod {
			return &ents[i], nil
		}
	}
	return nil, fmt.Errorf("entitlement not found for product %s", prod)
}

// SetProductEnabled toggles a product's enabled flag without allowing callers to
// escalate Tier or MonthlyQuota.
func (s *ServiceImpl) SetProductEnabled(ctx context.Context, teamID uint, prod entities.ProductSlug, enabled bool) (*entities.TeamEntitlement, error) {
	if !entities.IsValidProduct(prod) {
		return nil, fmt.Errorf("invalid product: %s", prod)
	}
	ent, err := s.GetEntitlement(ctx, teamID, prod)
	if err != nil {
		return nil, err
	}
	ent.Enabled = enabled
	if err := s.db.WithContext(ctx).Model(ent).Update("enabled", enabled).Error; err != nil {
		return nil, err
	}
	return ent, nil
}

// SyncSubscriptionFromStripe reconciles TeamSubscription, Team.PaymentPlan, and all
// 4 TeamEntitlement rows from a multi-item Stripe subscription.
func (s *ServiceImpl) SyncSubscriptionFromStripe(ctx context.Context, teamID uint, sub *stripe.Subscription) error {
	if s.db == nil || sub == nil {
		return nil
	}

	isEntitledStatus := sub.Status == stripe.SubscriptionStatusActive ||
		sub.Status == stripe.SubscriptionStatusTrialing ||
		sub.Status == stripe.SubscriptionStatusPastDue

	baseTier := entities.EntitlementTierFree
	type addonOverride struct {
		tier   entities.EntitlementTier
		itemID string
	}
	addons := make(map[entities.ProductSlug]addonOverride)

	if isEntitledStatus && sub.Items != nil {
		for _, item := range sub.Items.Data {
			isSuite, suiteTier, addonProd, addonTier := s.resolvePriceItem(item)
			if isSuite && entities.TierRank(suiteTier) > entities.TierRank(baseTier) {
				baseTier = suiteTier
			} else if addonProd != "" {
				prev, exists := addons[addonProd]
				if !exists || entities.TierRank(addonTier) > entities.TierRank(prev.tier) {
					addons[addonProd] = addonOverride{tier: addonTier, itemID: item.ID}
				}
			}
		}
	}

	var periodStart, periodEnd *time.Time
	if sub.CurrentPeriodStart > 0 {
		t := time.Unix(sub.CurrentPeriodStart, 0).UTC()
		periodStart = &t
	}
	if sub.CurrentPeriodEnd > 0 {
		t := time.Unix(sub.CurrentPeriodEnd, 0).UTC()
		periodEnd = &t
	}

	custID := ""
	if sub.Customer != nil {
		custID = sub.Customer.ID
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		subRec := entities.TeamSubscription{
			TeamID:               teamID,
			StripeCustomerID:     custID,
			StripeSubscriptionID: sub.ID,
			BaseTier:             baseTier,
			Status:               string(sub.Status),
			CurrentPeriodStart:   periodStart,
			CurrentPeriodEnd:     periodEnd,
			CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "team_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"stripe_customer_id",
				"stripe_subscription_id",
				"base_tier",
				"status",
				"current_period_start",
				"current_period_end",
				"cancel_at_period_end",
				"updated_at",
			}),
		}).Create(&subRec).Error; err != nil {
			return err
		}

		// Sync denormalized Team.PaymentPlan
		plan := entities.PaymentPlanFree
		switch baseTier {
		case entities.EntitlementTierEnterprise:
			plan = entities.PaymentPlanEnterprise
		case entities.EntitlementTierPro:
			plan = entities.PaymentPlanPro
		}
		teamUpdates := map[string]any{"payment_plan": plan}
		if custID != "" {
			teamUpdates["stripe_customer_id"] = custID
		}
		if err := tx.Model(&entities.Team{}).Where("id = ?", teamID).Updates(teamUpdates).Error; err != nil {
			return err
		}

		// Reconcile all 4 product entitlements
		resetAt := nextMonthReset(time.Now().UTC())
		if periodEnd != nil {
			resetAt = *periodEnd
		}
		for _, prod := range entities.AllProducts() {
			effectiveTier := baseTier
			source := entities.EntitlementSourceSuite
			var itemID *string

			if ov, ok := addons[prod]; ok && entities.TierRank(ov.tier) > entities.TierRank(baseTier) {
				effectiveTier = ov.tier
				source = entities.EntitlementSourceAddon
				idCopy := ov.itemID
				itemID = &idCopy
			}

			ent := entities.TeamEntitlement{
				TeamID:                   teamID,
				Product:                  prod,
				Tier:                     effectiveTier,
				Source:                   source,
				Enabled:                  true,
				StripeSubscriptionItemID: itemID,
				MonthlyQuota:             entities.DefaultMonthlyQuota(prod, effectiveTier),
				UsageResetAt:             &resetAt,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "team_id"}, {Name: "product"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"tier",
					"source",
					"stripe_subscription_item_id",
					"monthly_quota",
					"usage_reset_at",
					"updated_at",
				}),
			}).Create(&ent).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ServiceImpl) DowngradeTeamToFree(ctx context.Context, teamID uint) error {
	if s.db == nil {
		return nil
	}
	resetAt := nextMonthReset(time.Now().UTC())
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entities.Team{}).Where("id = ?", teamID).Update("payment_plan", entities.PaymentPlanFree).Error; err != nil {
			return err
		}
		_ = tx.Model(&entities.TeamSubscription{}).Where("team_id = ?", teamID).Updates(map[string]any{
			"base_tier":            entities.EntitlementTierFree,
			"status":               "canceled",
			"cancel_at_period_end": false,
		}).Error

		for _, prod := range entities.AllProducts() {
			if err := tx.Model(&entities.TeamEntitlement{}).
				Where("team_id = ? AND product = ?", teamID, prod).
				Updates(map[string]any{
					"tier":                        entities.EntitlementTierFree,
					"source":                      entities.EntitlementSourceSuite,
					"stripe_subscription_item_id": nil,
					"monthly_quota":               entities.DefaultMonthlyQuota(prod, entities.EntitlementTierFree),
					"usage_reset_at":              resetAt,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ServiceImpl) ResetPeriodUsage(ctx context.Context, teamID uint, nextResetAt time.Time) error {
	if s.db == nil {
		return nil
	}
	if nextResetAt.IsZero() {
		nextResetAt = nextMonthReset(time.Now().UTC())
	}
	return s.db.WithContext(ctx).
		Model(&entities.TeamEntitlement{}).
		Where("team_id = ?", teamID).
		Updates(map[string]any{
			"current_usage":  0,
			"usage_reset_at": nextResetAt.UTC(),
		}).Error
}

func (s *ServiceImpl) ResetExpiredFreeUsage(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	now := time.Now().UTC()
	next := nextMonthReset(now)
	return s.db.WithContext(ctx).
		Model(&entities.TeamEntitlement{}).
		Where("tier = ? AND usage_reset_at IS NOT NULL AND usage_reset_at <= ?", entities.EntitlementTierFree, now).
		Updates(map[string]any{
			"current_usage":  0,
			"usage_reset_at": next,
		}).Error
}
