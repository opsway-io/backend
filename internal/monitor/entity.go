package monitor

import (
	"time"

	"github.com/lib/pq"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/team"
)

type Monitor struct {
	ID        int
	Name      string          `gorm:"index;not null"`
	Tags      *pq.StringArray `gorm:"type:text[]"`
	Settings  Settings        `gorm:"not null;constraint:OnDelete:CASCADE"`
	TeamID    int             `gorm:"index;not null"`
	Team      team.Team
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (Monitor) TableName() string {
	return "monitors"
}

func (m *Monitor) SetTags(tags []string) {
	m.Tags = (*pq.StringArray)(&tags)
}

type Settings struct {
	ID        int
	Method    string         `gorm:"not null"`
	URL       string         `gorm:"not null"`
	Headers   *postgres.JSON `gorm:"type:jsonb"`
	Body      *[]byte        `gorm:"type:bytea"`
	Frequency time.Duration  `gorm:"not null"`
	MonitorID int            `gorm:"index;not null"`
	CreatedAt time.Time      `gorm:"index"`
	UpdatedAt time.Time      `gorm:"index"`
}

func (Settings) TableName() string {
	return "monitor_settings"
}
