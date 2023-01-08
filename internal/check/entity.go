package check

import (
	"time"

	"github.com/gofrs/uuid"
)

type Check struct {
	ID         uuid.UUID `gorm:"primary_key;type:UUID;default:generateUUIDv4()"`
	Method     string    `gorm:"index;not null"`
	URL        string    `gorm:"index;not null"`
	MonitorID  uint64    `gorm:"index;not null"`
	StatusCode uint64    `gorm:"index; not null"`
	Timing     Timing    `gorm:"embedded;embeddedPrefix:timing_"`
	TLS        *TLS      `gorm:"embedded;embeddedPrefix:tls_"`
	CreatedAt  time.Time `gorm:"index"`
}

func (Check) TableName() string {
	return "checks"
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
