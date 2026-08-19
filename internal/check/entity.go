package check

import (
	"time"

	"github.com/gofrs/uuid"
)

type Check struct {
	ID         uuid.UUID `gorm:"primary_key;type:UUID;default:generateUUIDv4()"`
	TeamID     uint64    `gorm:"index;not null"`
	Method     string    `gorm:"index;not null"`
	URL        string    `gorm:"index;not null"`
	Location   string    `gorm:"index;not null"`
	MonitorID  uint64    `gorm:"index;not null"`
	StatusCode uint64    `gorm:"index; not null"`
	Timing     Timing    `gorm:"embedded;embeddedPrefix:timing_"`
	TLS        *TLS      `gorm:"embedded;embeddedPrefix:tls_"`
	CreatedAt  time.Time `gorm:"index"`
}

// TableName returns the table name for the Check model, with ClickHouse engine options
func (Check) TableName() string {
	return "checks"
}

// TableOptions sets table options for ClickHouse to optimize group by queries
func (Check) TableOptions() string {
	return "ENGINE=MergeTree() ORDER BY (team_id, monitor_id, created_at)"
}

type Timing struct {
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	ContentTransfer  time.Duration
	Total            time.Duration `gorm:"index; not null"`
}

type TLS struct {
	Version   string
	Cipher    string
	Issuer    string
	Subject   string
	NotBefore time.Time
	NotAfter  time.Time
}
