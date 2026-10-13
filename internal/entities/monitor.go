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

	Settings MonitorSettings `gorm:"not null;constraint:OnDelete:CASCADE"`
	// Steps are executed in order (by Position) on every check.
	Steps []MonitorStep `gorm:"constraint:OnDelete:CASCADE"`
	// Assertions holds every assertion of the monitor (across all steps).
	// Each assertion references its step through StepID.
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

	Auth      MonitorSettingsAuth `gorm:"embedded;embeddedPrefix:auth_"`
	Locations []string            `gorm:"serializer:json"`

	UpdatedAt time.Time `gorm:"index"`
}

const (
	MonitorAuthMethodNone                    = "NONE"
	MonitorAuthMethodBasic                   = "BASIC"
	MonitorAuthMethodOAuth2ClientCredentials = "OAUTH2_CLIENT_CREDENTIALS"
)

type MonitorSettingsAuth struct {
	Method       string `gorm:"not null;default:'NONE'"`
	TokenURL     string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
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

// MonitorStep is a single HTTP request of a (multi-step) monitor.
type MonitorStep struct {
	ID        uint
	MonitorID uint `gorm:"index;not null"`
	Position  int  `gorm:"not null;default:0"`

	Name      string                  `gorm:"not null;default:''"`
	Method    string                  `gorm:"not null"`
	URL       string                  `gorm:"not null"`
	Headers   []MonitorSettingsHeader `gorm:"serializer:json"`
	Body      MonitorSettingsBody     `gorm:"embedded;embeddedPrefix:body_"`
	Variables []MonitorStepVariable   `gorm:"serializer:json"`

	Assertions []MonitorAssertion `gorm:"foreignKey:StepID;constraint:OnDelete:CASCADE"`

	CreatedAt time.Time
	UpdatedAt time.Time `gorm:"index"`
}

func (MonitorStep) TableName() string {
	return "monitor_steps"
}

// MonitorStepVariable extracts a value from a step's response, which later
// steps can reference as {{Name}}.
type MonitorStepVariable struct {
	Name     string
	Source   string
	Property string
}

type MonitorAssertion struct {
	ID        uint
	MonitorID uint  `gorm:"index;not null"`
	StepID    *uint `gorm:"index"`
	Position  int   `gorm:"not null;default:0"`

	Source   string `gorm:"not null"`
	Property string
	Operator string `gorm:"not null"`
	Target   string

	UpdatedAt time.Time `gorm:"index"`
}

func (MonitorAssertion) TableName() string {
	return "monitor_assertions"
}
