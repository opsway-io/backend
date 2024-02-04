package entities

import "time"

type Changelog struct {
	ID     uint
	TeamID uint `gorm:"index;not null"`

	Name    string           `gorm:"index;not null"`
	Entries []ChangelogEntry `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (Changelog) TableName() string {
	return "changelogs"
}

type ChangelogEntry struct {
	ID          uint
	ChangelogID uint `gorm:"index;not null"`

	Title   string `gorm:"index;not null"`
	Content string
	Authors []User `gorm:"many2many:changelog_entry_authors"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
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
