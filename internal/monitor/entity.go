package monitor

import (
	"time"

	"github.com/lib/pq"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/team"
)

type Monitor struct {
	ID         int
	Name       string         `gorm:"not null,index:idx_name"`
	Tags       pq.StringArray `gorm:"type:text[]"`
	SettingsID int
	Settings   Settings `gorm:"foreignKey:SettingsID"`
	TeamID     int      `gorm:"not null,index:idx_team_id"`
	Team       team.Team
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
