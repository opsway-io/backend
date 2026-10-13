package entities

import "time"

// ChannelType defines the medium for notification
type ChannelType string

const (
	ChannelEmail   ChannelType = "email"
	ChannelSMS     ChannelType = "sms"
	ChannelSlack   ChannelType = "slack"
	ChannelWebhook ChannelType = "webhook"
	ChannelVoice   ChannelType = "voice"
)

type UserNotificationRule struct {
	ID        uint
	UserID    uint        `gorm:"index;not null;uniqueIndex:idx_user_notification_rule"`
	Channel   ChannelType `gorm:"not null;uniqueIndex:idx_user_notification_rule"`
	Delay     int         `gorm:"not null;default:0;uniqueIndex:idx_user_notification_rule"` // Minutes to wait before notifying (0 = immediate)
	CreatedAt time.Time   `gorm:"index"`
	UpdatedAt time.Time   `gorm:"index"`
}

func (UserNotificationRule) TableName() string {
	return "user_notification_rules"
}
