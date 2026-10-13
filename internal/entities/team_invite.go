package entities

import "time"

// TeamInvite is a pending invitation of an email address to a team.
type TeamInvite struct {
	ID              uint
	TeamID          uint     `gorm:"uniqueIndex:team_invite_email;not null"`
	Email           string   `gorm:"uniqueIndex:team_invite_email;not null"`
	Role            TeamRole `gorm:"not null"`
	InvitedByUserID *uint

	Team *Team `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
}

func (TeamInvite) TableName() string {
	return "team_invites"
}
