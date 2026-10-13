package entities

import "time"

// UserNotificationRule describes how (channel) and when (delay in minutes) a user wants to be notified.
type UserNotificationRule struct {
	ID      uint
	UserID  uint   `gorm:"index;not null"`
	Channel string `gorm:"not null"`
	Delay   uint   `gorm:"not null;default:0"`

	User *User `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserNotificationRule) TableName() string {
	return "user_notification_rules"
}
