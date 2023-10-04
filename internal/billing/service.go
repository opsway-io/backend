package billing

import (
	"fmt"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/webhook"
)

type Config struct {
	SecretKey      string `mapstructure:"secret_key"`
	EndpointSecret string `mapstructure:"endpoint_secret"`
}

type Service interface {
	ConstructEvent(payload []byte, header string) (stripe.Event, error)
	CreateSession()
}

type ServiceImpl struct {
	Config Config
}

func NewService(conf Config) Service {

	stripe.Key = conf.SecretKey
	return &ServiceImpl{Config: conf}
}

func (s *ServiceImpl) ConstructEvent(payload []byte, header string) (stripe.Event, error) {
	fmt.Println(s.Config.SecretKey)
	fmt.Println(s.Config.EndpointSecret)
	return webhook.ConstructEvent(payload, header, s.Config.EndpointSecret)
}

func (s *ServiceImpl) CreateSession() {
	// priceId := "{{PRICE_ID}}"

	// params := &stripe.CheckoutSessionParams{
	// 	SuccessURL: "https://example.com/success.html?session_id={CHECKOUT_SESSION_ID}",
	// 	CancelURL:  "https://example.com/canceled.html",
	// 	Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
	// 	LineItems: []*stripe.CheckoutSessionLineItemParams{
	// 		&stripe.CheckoutSessionLineItemParams{
	// 			Price: stripe.String(priceId),
	// 			// For metered billing, do not pass quantity
	// 			Quantity: stripe.Int64(1),
	// 		},
	// 	},
	// }

	// s, _ := session.New(params)
}
