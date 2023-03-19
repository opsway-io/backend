package event

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	json "github.com/json-iterator/go"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	Publish(event events.Event) error
	// Subscribe(ctx context.Context, eventName string) (<-chan Event, error)
}

type service struct {
	publisher  *redisstream.Publisher
	subscriber *redisstream.Subscriber
}

func NewService(redisClient *redis.Client) (Service, error) {
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

	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client: redisClient,
		},
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

func (s *service) unmarshalEvent(data []byte) (events.Event, error) {
	var e events.Event

	err := json.Unmarshal(data, e)
	if err != nil {
		return nil, err
	}

	return e, nil
}
