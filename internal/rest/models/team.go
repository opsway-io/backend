package models

type Team struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Logo      string `json:"logo"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}
