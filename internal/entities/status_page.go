package entities

import (
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type StatusPageLayout string

const (
	StatusPageLayoutStats   StatusPageLayout = "STATS"
	StatusPageLayoutCompact StatusPageLayout = "COMPACT"
	StatusPageLayoutSimple  StatusPageLayout = "SIMPLE"
)

type StatusPage struct {
	ID     uint
	TeamID uint `gorm:"index;not null"`

	Name   string `gorm:"not null"`
	Domain string `gorm:"uniqueIndex;not null"`

	LogoURL              string `gorm:"not null;default:''"`
	LogoLink             string `gorm:"not null;default:''"`
	FaviconURL           string `gorm:"not null;default:''"`
	Layout               string `gorm:"not null;default:'STATS'"`
	CustomCSS            string `gorm:"not null;default:''"`
	HeaderHTML           string `gorm:"not null;default:''"`
	FooterHTML           string `gorm:"not null;default:''"`
	CustomComponentsHTML string `gorm:"not null;default:''"`
	ShowBranding         bool   `gorm:"not null;default:true"`
	IsPrivate            bool   `gorm:"not null;default:false"`
	PasswordHash         *string
	SupportURL           string `gorm:"not null;default:''"`

	// Ordered list of ungrouped monitor ids.
	MonitorIDs pq.Int64Array     `gorm:"type:bigint[];not null;default:'{}'"`
	Groups     []StatusPageGroup `gorm:"constraint:OnDelete:CASCADE"`

	Subscribers []StatusPageSubscriber `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (StatusPage) TableName() string {
	return "status_pages"
}

func (s *StatusPage) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	h := string(hash)
	s.PasswordHash = &h

	return nil
}

func (s *StatusPage) CheckPassword(password string) bool {
	if s.PasswordHash == nil || password == "" {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(*s.PasswordHash), []byte(password)) == nil
}

// AllMonitorIDs returns the ungrouped and grouped monitor ids, de-duplicated, in display order.
func (s *StatusPage) AllMonitorIDs() []uint {
	seen := map[int64]struct{}{}
	ids := []uint{}

	add := func(list pq.Int64Array) {
		for _, id := range list {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, uint(id))
		}
	}

	add(s.MonitorIDs)
	for _, g := range s.Groups {
		add(g.MonitorIDs)
	}

	return ids
}

type StatusPageGroup struct {
	ID           uint
	StatusPageID uint `gorm:"index;not null"`

	Name       string        `gorm:"not null"`
	Order      int           `gorm:"not null;default:0"`
	MonitorIDs pq.Int64Array `gorm:"type:bigint[];not null;default:'{}'"`
}

func (StatusPageGroup) TableName() string {
	return "status_page_groups"
}

type StatusPageSubscriberChannel string

const (
	StatusPageSubscriberChannelEmail   StatusPageSubscriberChannel = "EMAIL"
	StatusPageSubscriberChannelWebhook StatusPageSubscriberChannel = "WEBHOOK"
	StatusPageSubscriberChannelSlack   StatusPageSubscriberChannel = "SLACK"
)

type StatusPageSubscriber struct {
	ID           uint
	StatusPageID uint `gorm:"index;not null;uniqueIndex:status_page_subscriber_unique"`

	Channel     StatusPageSubscriberChannel `gorm:"not null;uniqueIndex:status_page_subscriber_unique"`
	Email       string                      `gorm:"not null;uniqueIndex:status_page_subscriber_unique"`
	Destination string                      `gorm:"not null;default:'';uniqueIndex:status_page_subscriber_unique"`

	VerificationToken string `gorm:"uniqueIndex;not null"`
	VerifiedAt        *time.Time

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (StatusPageSubscriber) TableName() string {
	return "status_page_subscribers"
}
