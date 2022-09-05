package models

import "github.com/opsway-io/backend/internal/user"

type User struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
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
