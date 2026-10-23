package entities

import (
	"time"
)

type ServiceCatalogItem struct {
	ID                uint              `gorm:"primaryKey" json:"id"`
	TeamID            uint              `gorm:"uniqueIndex:idx_team_service_slug;not null" json:"teamId"`
	Slug              string            `gorm:"uniqueIndex:idx_team_service_slug;not null" json:"slug"`
	Name              string            `gorm:"not null" json:"name"`
	Description       string            `json:"description"`
	LogSearchServices []string          `gorm:"serializer:json" json:"logSearchServices"`
	MonitorIDs        []uint            `gorm:"serializer:json" json:"monitorIds"`
	RepositoryURL     *string           `json:"repositoryUrl,omitempty"`
	PantomathProduct  *string           `json:"pantomathProduct,omitempty"`
	EnvironmentLabels map[string]string `gorm:"serializer:json" json:"environmentLabels"`
	CreatedAt         time.Time         `gorm:"index" json:"createdAt"`
	UpdatedAt         time.Time         `gorm:"index" json:"updatedAt"`
}

func (ServiceCatalogItem) TableName() string {
	return "service_catalog_items"
}
