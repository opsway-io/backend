package entities

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
	"github.com/opsway-io/backend/internal/connectors/postgres"
)

type Monitor struct {
	ID        uint
	Name      string          `gorm:"index;not null"`
	Tags      *pq.StringArray `gorm:"type:text[]"`
	Settings  MonitorSettings `gorm:"not null;constraint:OnDelete:CASCADE"`
	Incidents []Incident      `gorm:"constraint:OnDelete:CASCADE"`
	TeamID    uint            `gorm:"index;not null"`
	CreatedAt time.Time       `gorm:"index"`
	UpdatedAt time.Time       `gorm:"index"`
}

func (Monitor) TableName() string {
	return "monitors"
}

func (m *Monitor) SetTags(tags []string) {
	m.Tags = (*pq.StringArray)(&tags)
}

func (m *Monitor) GetTags() []string {
	if m.Tags == nil {
		return []string{}
	}

	return []string(*m.Tags)
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

func (m *Monitor) SetHeaders(headers map[string]string) error {
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
