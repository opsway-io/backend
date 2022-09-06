package models

import "github.com/opsway-io/backend/internal/user"

type User struct {
	ID          int    `json:"id" validate:"required,numeric,gte=0"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	DisplayName string `json:"displayName" validate:"required,min=1,max=255"`
	Email       string `json:"email" validate:"required,email"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

func UserToResponse(u user.User) User {
	return User{
		ID:          u.ID,
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Email:       u.Email,
		CreatedAt:   u.CreatedAt.Unix(),
		UpdatedAt:   u.UpdatedAt.Unix(),
	}
}

func UsersToResponse(us []user.User) []User {
	users := make([]User, len(us))
	for i, u := range us {
		users[i] = UserToResponse(u)
	}

	return users
}

func RequestToUser(u User) user.User {
	return user.User{
		ID:          u.ID,
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Email:       u.Email,
	}
}
