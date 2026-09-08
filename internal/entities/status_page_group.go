package entities

type StatusPageGroup struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	StatusPageID uint      `gorm:"index;not null;constraint:OnDelete:CASCADE" json:"statusPageId"`
	Name         string    `gorm:"not null" json:"name"`
	Order        int       `gorm:"default:0" json:"order"`
	Monitors     []Monitor `gorm:"many2many:status_page_group_monitors;constraint:OnDelete:CASCADE" json:"monitors"`
}

func (StatusPageGroup) TableName() string {
	return "status_page_groups"
}
