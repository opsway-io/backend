package billing

import (
	"os"
	"strconv"

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
	"github.com/stripe/stripe-go/v81/webhook"
)

type Config struct {
	PublishableKey string `mapstructure:"publishable_key"`
	SecretKey      string `mapstructure:"secret_key"`
	WebhookSecret  string `mapstructure:"webhook_secret"`
	Domain         string `mapstructure:"domain"`
}

type Service interface {
	PostConfig() StripeConfig
	CreateCheckoutSession(team *entities.Team, priceLookupKey string) (*stripe.CheckoutSession, error)
	UpdateSubscribtion(team *entities.Team, priceLookupKey string) (*stripe.Subscription, error)
	CancelSubscribtion(team *entities.Team) (*stripe.Subscription, error)
	GetCheckoutSession(sessionID string) (*stripe.CheckoutSession, error)
	GetLineItems(sessionID string) *session.LineItemIter
	GetPrices(priceLookupKeys []string) ([]*stripe.Price, error)
	GetCustomerSubscribtion(customerID string) *subscription.Iter
	GetSubscribtion(subID string) (*stripe.Subscription, error)
	GetProduct(productID string) (*stripe.Product, error)
	GetProducts() *product.Iter
	CreateCustomerPortal(team *entities.Team) (*stripe.BillingPortalSession, error)
	ConstructEvent(payload []byte, header string) (stripe.Event, error)
	GetCustomerSession(team *entities.Team) (*stripe.CustomerSession, error)
	GetBillingPortal(team *entities.Team) (*stripe.BillingPortalConfiguration, error)
}

type ServiceImpl struct {
	Config Config
}

func NewService(conf Config) Service {

	stripe.Key = conf.SecretKey

	return &ServiceImpl{Config: conf}
}

type StripeConfig struct {
	PublishableKey string `json:"publishableKey"`
	BasicPrice     string `json:"basicPrice"`
	ProPrice       string `json:"proPrice"`
}

func (s *ServiceImpl) PostConfig() StripeConfig {
	return StripeConfig{
		PublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		BasicPrice:     os.Getenv("BASIC_PRICE_ID"),
		ProPrice:       os.Getenv("PRO_PRICE_ID"),
	}
}

func (s *ServiceImpl) CreateCheckoutSession(team *entities.Team, priceLookupKey string) (*stripe.CheckoutSession, error) {
	priceID, err := s.GetPrices([]string{priceLookupKey})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get price")
	}

	params := &stripe.CheckoutSessionParams{
		SuccessURL:        stripe.String(s.Config.Domain + "/team/subscription"),
		CancelURL:         stripe.String(s.Config.Domain + "/team/subscription"),
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		ClientReferenceID: stripe.String(strconv.FormatUint(uint64(team.ID), 10)),

		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{

				Price:    stripe.String(priceID[0].ID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	if team.StripeCustomerID != nil {
		params.Customer = stripe.String(*team.StripeCustomerID)
	}

	return session.New(params)
}

func (s *ServiceImpl) UpdateSubscribtion(team *entities.Team, priceLookupKey string) (*stripe.Subscription, error) {
	// Set Customer on session if already a customer

	priceID, err := s.GetPrices([]string{priceLookupKey})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get price")
	}
	sub := s.GetCustomerSubscribtion(*team.StripeCustomerID)
	sub.Next()
	teamSubscription := sub.Subscription()

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(teamSubscription.Items.Data[0].ID),
				Price: stripe.String(priceID[0].ID),
			},
		},
	}
	result, err := subscription.Update(teamSubscription.ID, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update subscription")
	}

	return result, nil
}

func (s *ServiceImpl) CancelSubscribtion(team *entities.Team) (*stripe.Subscription, error) {
	// Set Customer on session if already a customer

	sub := s.GetCustomerSubscribtion(*team.StripeCustomerID)
	sub.Next()
	teamSubscription := sub.Subscription()

	params := &stripe.SubscriptionCancelParams{}
	result, err := subscription.Cancel(teamSubscription.ID, params)

	if err != nil {
		return nil, errors.Wrap(err, "failed to update subscription")
	}

	return result, nil
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

		LookupKeys: stripe.StringSlice(
			priceLookupKeys,
		),
	}
	i := price.List(params)

	prices := make([]*stripe.Price, 0)
	for i.Next() {
		p := i.Price()
		prices = append(prices, p)
	}

	return prices, nil
}

func (s *ServiceImpl) GetCustomerSubscribtion(customerID string) *subscription.Iter {
	params := &stripe.SubscriptionListParams{Customer: stripe.String(customerID)}
	return subscription.List(params)
}

func (s *ServiceImpl) GetSubscribtion(subID string) (*stripe.Subscription, error) {
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
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(*team.StripeCustomerID),
		ReturnURL: stripe.String(s.Config.Domain),
	}
	return portalsession.New(params)
}

func (s *ServiceImpl) ConstructEvent(payload []byte, header string) (stripe.Event, error) {
	return webhook.ConstructEventWithOptions(payload, header, s.Config.WebhookSecret, webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
}

func (s *ServiceImpl) GetCustomerSession(team *entities.Team) (*stripe.CustomerSession, error) {
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
