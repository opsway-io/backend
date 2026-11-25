package entities

import (
	"time"
)

// UsageDailyCounter tracks lightweight per-team daily event counts (e.g. MCP Gateway tool calls)
// so the Usage page can display historical trends without scanning raw logs.
type UsageDailyCounter struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	TeamID    uint        `gorm:"uniqueIndex:idx_usage_daily_key;not null" json:"teamId"`
	Date      string      `gorm:"uniqueIndex:idx_usage_daily_key;size:10;not null" json:"date"` // YYYY-MM-DD (UTC)
	Product   ProductSlug `gorm:"uniqueIndex:idx_usage_daily_key;size:32;not null" json:"product"`
	MetricKey string      `gorm:"uniqueIndex:idx_usage_daily_key;size:64;not null" json:"metricKey"`
	Count     int64       `gorm:"not null;default:0" json:"count"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

func (UsageDailyCounter) TableName() string {
	return "usage_daily_counters"
}
