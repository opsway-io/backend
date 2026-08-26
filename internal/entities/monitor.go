package entities

import (
	"time"
)

type MonitorState int

const (
	MonitorStateInactive    MonitorState = 0
	MonitorStateActive      MonitorState = 1
	MonitorStateMaintenance MonitorState = 2
)

type Monitor struct {
	ID     uint `json:"id"`
	TeamID uint `gorm:"index;not null" json:"teamId"`

	State MonitorState `gorm:"not null;default:0" json:"state"`
	Name  string       `gorm:"index;not null" json:"name"`

	Settings   MonitorSettings    `gorm:"not null;constraint:OnDelete:CASCADE" json:"settings"`
	Assertions []MonitorAssertion `gorm:"constraint:OnDelete:CASCADE" json:"assertions"`
	Incidents  []Incident         `gorm:"constraint:OnDelete:CASCADE" json:"incidents"`

	CreatedAt time.Time `gorm:"index" json:"createdAt"`
	UpdatedAt time.Time `gorm:"index" json:"updatedAt"`
}

func (Monitor) TableName() string {
	return "monitors"
}

func (m *Monitor) GetStateString() string {
	switch m.State {
	case MonitorStateInactive:
		return "INACTIVE"
	case MonitorStateActive:
		return "ACTIVE"
	case MonitorStateMaintenance:
		return "MAINTENANCE"
	default:
		return "UNKNOWN"
	}
}

func (m *Monitor) SetStateString(state string) error {
	switch state {
	case "INACTIVE":
		m.State = MonitorStateInactive
	case "ACTIVE":
		m.State = MonitorStateActive
	case "MAINTENANCE":
		m.State = MonitorStateMaintenance
	}

	return nil
}

func GetMonitorStateEnumFromString(state string) MonitorState {
	switch state {
	case "INACTIVE":
		return MonitorStateInactive
	case "ACTIVE":
		return MonitorStateActive
	case "MAINTENANCE":
		return MonitorStateMaintenance
	default:
		return -1
	}
}

type MonitorSettings struct {
	ID        uint
	MonitorID uint `gorm:"uniqueIndex;not null"`

	Method    string        `gorm:"not null"`
	URL       string        `gorm:"not null"`
	Frequency time.Duration `gorm:"not null;serializer:timeDurationSeconds"`

	Headers   []MonitorSettingsHeader `gorm:"serializer:json"`
	Body      MonitorSettingsBody     `gorm:"embedded;embeddedPrefix:body_"`
	TLS       MonitorSettingsTLS      `gorm:"embedded;embeddedPrefix:tls_"`
	Locations []string                `gorm:"serializer:json"`

	UpdatedAt time.Time `gorm:"index"`
}

type MonitorSettingsHeader struct {
	Key   string `gorm:"not null"`
	Value string `gorm:"not null"`
}

type MonitorSettingsBody struct {
	Content *[]byte `gorm:"type:bytea"`
	Type    string  `gorm:"not null;default:'NONE'"`
}

func (m *MonitorSettingsBody) SetContentString(body *string) {
	if body == nil {
		m.Content = nil

		return
	}

	b := []byte(*body)
	m.Content = &b
}

func (m *MonitorSettingsBody) GetContentString() *string {
	if m.Content == nil {
		return nil
	}

	body := string(*m.Content)

	return &body
}

type MonitorSettingsTLS struct {
	Enabled                 bool  `gorm:"not null;default:false"`
	VerifyHostname          *bool `gorm:"default:null"`
	CheckExpiration         *bool `gorm:"default:null"`
	ExpirationThresholdDays *uint `gorm:"default:null"`
}

func (MonitorSettings) TableName() string {
	return "monitor_settings"
}

func (ms *MonitorSettings) GetFrequencySeconds() uint64 {
	return uint64(ms.Frequency / time.Second)
}

func (ms *MonitorSettings) SetFrequencySeconds(seconds uint64) {
	ms.Frequency = time.Duration(seconds) * time.Second
}

type MonitorAssertion struct {
	ID        uint
	MonitorID uint `gorm:"index;not null"`

	Source   string `gorm:"not null"`
	Property string
	Operator string `gorm:"not null"`
	Target   string

	UpdatedAt time.Time `gorm:"index"`
}

func (MonitorAssertion) TableName() string {
	return "monitor_assertions"
}
