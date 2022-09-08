package models

import "github.com/opsway-io/backend/internal/entities"

type User struct {
	ID          uint   `json:"id" validate:"numeric,gte=0"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	DisplayName string `json:"displayName" validate:"required,min=1,max=255"`
	Email       string `json:"email" validate:"required,email"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

func UserToResponse(u entities.User) User {
	return User{
		ID:          u.ID,
		Name:        u.Name,
		DisplayName: *u.DisplayName,
		Email:       u.Email,
		CreatedAt:   u.CreatedAt.Unix(),
		UpdatedAt:   u.UpdatedAt.Unix(),
	}
}

func UsersToResponse(us []entities.User) []User {
	users := make([]User, len(us))
	for i, u := range us {
		users[i] = UserToResponse(u)
	}

	return users
}

func RequestToUser(u User) entities.User {
	return entities.User{
		ID:          u.ID,
		Name:        u.Name,
		DisplayName: &u.DisplayName,
		Email:       u.Email,
	}
}
