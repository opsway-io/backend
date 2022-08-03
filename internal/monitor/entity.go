package monitor

import (
	"time"

	"github.com/lib/pq"
	"github.com/opsway-io/backend/internal/connectors/postgres"
)

type Monitor struct {
	ID         int
	TeamID     int            `gorm:"not null,index"`
	Name       string         `gorm:"not null"`
	Tags       pq.StringArray `gorm:"type:text[]"`
	SettingsID int
	Settings   Settings `gorm:"foreignKey:SettingsID"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (Monitor) TableName() string {
	return "monitors"
}

type Settings struct {
	ID        int
	Method    string        `gorm:"not null"`
	URL       string        `gorm:"not null"`
	Headers   postgres.JSON `gorm:"type:jsonb"`
	Body      []byte        `gorm:"type:bytea"`
	Interval  time.Duration `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Settings) TableName() string {
	return "monitor_settings"
}
