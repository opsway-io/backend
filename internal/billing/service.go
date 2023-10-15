package billing

import (
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/checkout/session"
	"github.com/stripe/stripe-go/webhook"
)

type Config struct {
	PublishableKey string `mapstructure:"publishable_key"`
	SecretKey      string `mapstructure:"secret_key"`
	WebhookSecret  string `mapstructure:"webhook_secret"`
	Domain         string `mapstructure:"domain"`
}

type Service interface {
	ConstructEvent(payload []byte, header string) (stripe.Event, error)
	CreateSession(priceId string) (*stripe.CheckoutSession, error)
}

type ServiceImpl struct {
	Config Config
}

func NewService(conf Config) Service {

	stripe.Key = conf.SecretKey

	return &ServiceImpl{Config: conf}
}

func (s *ServiceImpl) ConstructEvent(payload []byte, header string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, header, s.Config.WebhookSecret)
}

func (s *ServiceImpl) CreateSession(priceId string) (*stripe.CheckoutSession, error) {
	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(s.Config.Domain + "/success.html?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(s.Config.Domain + "/canceled.html"),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Name:     stripe.String(priceId),
				Quantity: stripe.Int64(1),
			},
		},
	}

	return session.New(params)
}
