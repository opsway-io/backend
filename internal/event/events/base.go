package events

import (
	"github.com/vmihailenco/msgpack"
)

type Event interface {
	Name() string
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

type BaseEvent struct{}

func (e *BaseEvent) Name() string {
	return ""
}

func (e *BaseEvent) Marshal() ([]byte, error) {
	b, err := msgpack.Marshal(e)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (e *BaseEvent) Unmarshal(data []byte) error {
	return msgpack.Unmarshal(data, e)
}
