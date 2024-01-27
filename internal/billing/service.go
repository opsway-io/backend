package billing

import (
	"os"
	"strconv"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	"github.com/stripe/stripe-go/v76/checkout/session"
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

func (s *ServiceImpl) CreateCheckoutSession(team *entities.Team, priceLookupKey string) (*stripe.CheckoutSession, error) {
	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String("https://my.opsway.io/team/plan"),
		// ReturnURL:         stripe.String("https://my.opsway.io/team/plan"),
		CancelURL:         stripe.String(s.Config.Domain + "/canceled.html"),
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		ClientReferenceID: stripe.String(strconv.FormatUint(uint64(team.ID), 10)),

		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceLookupKey),
				Quantity: stripe.Int64(1),
			},
		},
	}

	// Set Customer on session if already a customer
	if team.StripeCustomerID != nil {
		params.Customer = stripe.String(*team.StripeCustomerID)
	}

	return session.New(params)
}

func (s *ServiceImpl) GetCheckoutSession(sessionID string) (*stripe.CheckoutSession, error) {
	return session.Get(sessionID, &stripe.CheckoutSessionParams{})
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
