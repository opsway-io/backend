package billing

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/price"
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
	CreateCheckoutSession(priceId string) (*stripe.CheckoutSession, error)
	GetCheckoutSession(sessionID string) (*stripe.CheckoutSession, error)
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

func (s *ServiceImpl) CreateCheckoutSession(lookupKey string) (*stripe.CheckoutSession, error) {
	priceParams := &stripe.PriceListParams{
		LookupKeys: stripe.StringSlice([]string{
			lookupKey,
		}),
	}

	i := price.List(priceParams)
	var price *stripe.Price
	for i.Next() {
		p := i.Price()
		price = p
	}

	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(s.Config.Domain + "/success.html?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(s.Config.Domain + "/canceled.html"),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(price.ID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	return session.New(params)
}

func (s *ServiceImpl) GetCheckoutSession(sessionID string) (*stripe.CheckoutSession, error) {
	return session.Get(sessionID, nil)
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
	return webhook.ConstructEvent(payload, header, s.Config.WebhookSecret)
}
