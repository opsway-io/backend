package billing

import (
	"os"
	"strconv"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/price"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"
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
	GetPrice(priceLookupKey string) (*stripe.Price, error)
	GetCustomerSubscribtion(customerID string) *subscription.Iter
	GetSubscribtion(subID string) (*stripe.Subscription, error)
	CreateCustomerPortal(sessionID string) (*stripe.BillingPortalSession, error)
	ConstructEvent(payload []byte, header string) (stripe.Event, error)
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
	priceID, err := s.GetPrice(priceLookupKey)
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

				Price:    stripe.String(priceID.ID),
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

	priceID, err := s.GetPrice(priceLookupKey)
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
				Price: stripe.String(priceID.ID),
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

func (s *ServiceImpl) GetPrice(priceLookupKey string) (*stripe.Price, error) {
	params := &stripe.PriceListParams{
		LookupKeys: stripe.StringSlice([]string{
			priceLookupKey,
		}),
	}
	i := price.List(params)

	var price *stripe.Price
	for i.Next() {
		p := i.Price()
		price = p
	}
	if price == nil {
		return nil, errors.New("Price not found for lookup key" + priceLookupKey)
	}
	return price, nil
}

func (s *ServiceImpl) GetCustomerSubscribtion(customerID string) *subscription.Iter {
	params := &stripe.SubscriptionListParams{Customer: stripe.String(customerID)}
	return subscription.List(params)
}

func (s *ServiceImpl) GetSubscribtion(subID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{}
	return subscription.Get(subID, params)

}

func (s *ServiceImpl) CreateCustomerPortal(sessionID string) (*stripe.BillingPortalSession, error) {
	// For demonstration purposes, we're using the Checkout session to retrieve the customer ID.
	// Typically this is stored alongside the authenticated user in your database.
	se, err := session.Get(sessionID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get session")
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(se.Customer.ID),
		ReturnURL: stripe.String(s.Config.Domain),
	}
	return portalsession.New(params)
}

func (s *ServiceImpl) ConstructEvent(payload []byte, header string) (stripe.Event, error) {
	return webhook.ConstructEventWithOptions(payload, header, s.Config.WebhookSecret, webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
}
