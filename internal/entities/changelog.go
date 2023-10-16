package entities

import "time"

type Changelog struct {
	ID        uint
	Name      string           `gorm:"index;not null"`
	TeamID    uint             `gorm:"index;not null"`
	Entries   []ChangelogEntry `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt time.Time        `gorm:"index"`
	UpdatedAt time.Time        `gorm:"index"`
}

func (Changelog) TableName() string {
	return "changelogs"
}

type ChangelogEntry struct {
	ID          uint
	Title       string `gorm:"index;not null"`
	Content     string
	ChangelogID uint      `gorm:"index;not null"`
	Authors     []User    `gorm:"many2many:changelog_entry_authors"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time `gorm:"index"`
}

func (ChangelogEntry) TableName() string {
	return "changelog_entries"
}

type ChangelogEntryAuthor struct {
	ID               uint
	ChangelogEntryID uint `gorm:"index;not null"`
	UserID           uint `gorm:"index;not null"`
}

func (ChangelogEntryAuthor) TableName() string {
	return "changelog_entry_authors"
}
