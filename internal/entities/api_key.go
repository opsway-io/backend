package entities

import "time"

// ApiKey is a team scoped API key. Only a SHA-256 hash of the key is stored.
type ApiKey struct {
	ID              uint
	TeamID          uint   `gorm:"index;not null"`
	CreatedByUserID uint   `gorm:"index;not null"`
	Name            string `gorm:"not null"`
	Prefix          string `gorm:"not null"`
	Hash            string `gorm:"uniqueIndex;not null"`
	LastUsedAt      *time.Time

	Team          *Team `gorm:"constraint:OnDelete:CASCADE"`
	CreatedByUser *User `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
}

func (ApiKey) TableName() string {
	return "api_keys"
}
