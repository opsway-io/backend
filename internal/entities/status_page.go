package entities

import (
	"time"
)

type StatusPage struct {
	ID        uint      `gorm:"primaryKey"`
	TeamID    uint      `gorm:"index;not null"`
	Name      string    `gorm:"not null"`
	Domain    string    `gorm:"uniqueIndex;not null"`
	LogoURL      string    `gorm:"default:''"`
	LogoLink     string    `gorm:"default:''"`
	FaviconURL   string    `gorm:"default:''"`
	Layout       string    `gorm:"default:'STATS'"`
	CustomCSS            string    `gorm:"default:''"`
	HeaderHTML           string    `gorm:"default:''"`
	FooterHTML           string    `gorm:"default:''"`
	CustomComponentsHTML string    `gorm:"default:''"`
	ShowBranding bool      `gorm:"default:true"`
	IsPrivate    bool      `gorm:"default:false"`
	PasswordHash string    `gorm:"default:''"`

	Monitors  []Monitor `gorm:"many2many:status_page_monitors;constraint:OnDelete:CASCADE"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (StatusPage) TableName() string {
	return "status_pages"
}
