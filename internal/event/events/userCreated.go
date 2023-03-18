package events

import "github.com/opsway-io/backend/internal/entities"

const UserCreated = "user.created"

type UserCreatedData struct {
	BaseEvent
	entities.User
}

func NewUserCreatedEvent(user *entities.User) *UserCreatedData {
	return &UserCreatedData{
		User: *user,
	}
}

func (e *UserCreatedData) Name() string {
	return UserCreated
}
