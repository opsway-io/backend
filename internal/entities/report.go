package entities

import (
	"errors"
	"time"

	"github.com/opsway-io/backend/internal/check"
	"gorm.io/datatypes"
)

type ReportType string

const (
	ReportTypeUptime      ReportType = "UPTIME"
	ReportTypePerformance ReportType = "PERFORMANCE"
	ReportTypeIncident    ReportType = "INCIDENT"
	ReportTypeAll         ReportType = "ALL"
	ReportTypeCustom      ReportType = "CUSTOM"
)

type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "PENDING"
	ReportStatusCompleted ReportStatus = "COMPLETED"
	ReportStatusFailed    ReportStatus = "FAILED"
)

type Report struct {
	ID     uint
	TeamID uint         `gorm:"index;not null"`
	Type   ReportType   `gorm:"index;not null"`
	Status ReportStatus `gorm:"index;not null;default:COMPLETED"`
	Start  *time.Time
	End    *time.Time
	Report datatypes.JSONType[ReportData] `gorm:"not null"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (Report) TableName() string {
	return "reports"
}

type ReportData struct {
	Uptime      *[]check.MonitorUptime      `json:"uptime,omitempty"`
	Performance *[]check.MonitorPerformance `json:"performance,omitempty"`
	Incident    *[]ReportMonitorIncident    `json:"incident,omitempty"`
	All         *string                     `json:"all,omitempty"`
	Custom      *string                     `json:"custom,omitempty"`
}

type ReportMonitorIncident struct {
	MonitorID uint `json:"monitorId"`
	Count     int  `json:"count"`
}

func ReportFrom(source any) (ReportType, error) {
	s, ok := source.(string)
	if !ok {
		return "", errors.New("invalid report type, must be string")
	}

	switch s {
	case "UPTIME":
		return ReportTypeUptime, nil
	case "PERFORMANCE":
		return ReportTypePerformance, nil
	case "INCIDENT":
		return ReportTypeIncident, nil
	case "ALL":
		return ReportTypeAll, nil
	case "CUSTOM":
		return ReportTypeCustom, nil
	default:
		return "", errors.New("invalid report type")
	}
}
