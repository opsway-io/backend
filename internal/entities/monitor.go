package entities

import (
	"time"

	json "github.com/json-iterator/go"

	"github.com/opsway-io/backend/internal/connectors/postgres"
)

type MonitorState int

const (
	MonitorStateInactive MonitorState = 0
	MonitorStateActive   MonitorState = 1
)

type Monitor struct {
	ID         uint
	State      MonitorState       `gorm:"not null;default:0"`
	Name       string             `gorm:"index;not null"`
	Settings   MonitorSettings    `gorm:"not null;constraint:OnDelete:CASCADE"`
	Assertions []MonitorAssertion `gorm:"constraint:OnDelete:CASCADE"`
	Incidents  []Incident         `gorm:"constraint:OnDelete:CASCADE"`
	TeamID     uint               `gorm:"index;not null"`
	CreatedAt  time.Time          `gorm:"index"`
	UpdatedAt  time.Time          `gorm:"index"`
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

func (m *Monitor) SetStateFromString(state string) error {
	switch state {
	case "INACTIVE":
		m.State = MonitorStateInactive
	case "ACTIVE":
		m.State = MonitorStateActive
	}

	return nil
}

func (m *Monitor) SetBodyStr(body string) {
	byts := []byte(body)
	m.Settings.Body = &byts
}

func (m *Monitor) GetBodyStr() *string {
	if m.Settings.Body == nil {
		return nil
	}

	body := string(*m.Settings.Body)

	return &body
}

func (m *Monitor) SetHeaders(headers []struct{}) error {
	byts, err := json.Marshal(headers)
	if err != nil {
		return err
	}

	m.Settings.Headers = (*postgres.JSON)(&byts)

	return nil
}

func (m *Monitor) GetHeaders() (map[string]string, error) {
	if m.Settings.Headers == nil {
		return nil, nil
	}

	headers := map[string]string{}
	err := json.Unmarshal([]byte(*m.Settings.Headers), &headers)
	if err != nil {
		return nil, err
	}

	return headers, nil
}

type MonitorSettings struct {
	ID        uint
	Method    string         `gorm:"not null"`
	URL       string         `gorm:"not null"`
	Headers   *postgres.JSON `gorm:"type:jsonb"`
	Body      *[]byte        `gorm:"type:bytea"`
	BodyType  string         `gorm:"not null"`
	Frequency time.Duration  `gorm:"not null"`
	MonitorID int            `gorm:"index;not null"`
	CreatedAt time.Time      `gorm:"index"`
	UpdatedAt time.Time      `gorm:"index"`
}

func (MonitorSettings) TableName() string {
	return "monitor_settings"
}

func (ms *MonitorSettings) SetHeaders(headers map[string]string) error {
	b, err := json.Marshal(headers)
	if err != nil {
		return err
	}

	ms.Headers = (*postgres.JSON)(&b)

	return nil
}

func (ms *MonitorSettings) GetHeaders() (map[string]string, error) {
	if ms.Headers == nil {
		return nil, nil
	}

	var headers map[string]string
	if err := json.Unmarshal(*ms.Headers, &headers); err != nil {
		return nil, err
	}

	return headers, nil
}

func (ms *MonitorSettings) GetFrequencySeconds() uint64 {
	return uint64(ms.Frequency / time.Second)
}

func (ms *MonitorSettings) SetFrequencySeconds(frequency uint64) {
	ms.Frequency = time.Duration(frequency) * time.Second
}

type MonitorAssertion struct {
	ID        uint
	MonitorID uint   `gorm:"index;not null"`
	Source    string `gorm:"not null"`
	Property  string
	Operator  string `gorm:"not null"`
	Target    string
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (MonitorAssertion) TableName() string {
	return "monitor_assertions"
}
