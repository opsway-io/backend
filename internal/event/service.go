package event

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	json "github.com/json-iterator/go"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	Publish(event events.Event) error
	Subscribe(ctx context.Context, eventName string) (<-chan *message.Message, error)
}

type service struct {
	publisher  *redisstream.Publisher
	subscriber *redisstream.Subscriber
}

func NewService(redisClient *redis.Client, consumerGroup string) (Service, error) {
	publisher, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client:     redisClient,
			Marshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return nil, err
	}

	subscriberConfig := redisstream.SubscriberConfig{
		Client: redisClient,
	}
	if consumerGroup != "" {
		subscriberConfig.ConsumerGroup = consumerGroup
		subscriberConfig.Consumer = watermill.NewShortUUID()
	}

	subscriber, err := redisstream.NewSubscriber(
		subscriberConfig,
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return nil, err
	}

	return &service{
		publisher:  publisher,
		subscriber: subscriber,
	}, nil
}

func (s *service) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return s.subscriber.Subscribe(ctx, topic)
}

func (s *service) Publish(event events.Event) error {
	byts, err := s.marshal(event)
	if err != nil {
		return err
	}

	return s.publisher.Publish(string(event.Name()), &message.Message{
		Payload: byts,
	})
}

func (s *service) marshal(e events.Event) ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return b, nil
}
