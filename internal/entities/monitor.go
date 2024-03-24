package entities

import (
	"time"
)

type MonitorState int

const (
	MonitorStateInactive MonitorState = 0
	MonitorStateActive   MonitorState = 1
)

type Monitor struct {
	ID     uint
	TeamID uint `gorm:"index;not null"`

	State MonitorState `gorm:"not null;default:0"`
	Name  string       `gorm:"index;not null"`

	Settings   MonitorSettings    `gorm:"not null;constraint:OnDelete:CASCADE"`
	Assertions []MonitorAssertion `gorm:"constraint:OnDelete:CASCADE"`
	Incidents  []Incident         `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
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
	}

	return nil
}

func GetMonitorStateEnumFromString(state string) MonitorState {
	switch state {
	case "INACTIVE":
		return MonitorStateInactive
	case "ACTIVE":
		return MonitorStateActive
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

	Headers []MonitorSettingsHeader `gorm:"serializer:json"`
	Body    MonitorSettingsBody     `gorm:"embedded;embeddedPrefix:body_"`
	TLS     MonitorSettingsTLS      `gorm:"embedded;embeddedPrefix:tls_"`

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
