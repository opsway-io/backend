package logger

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// LogMessage matches the struct expected by log-search
type LogMessage struct {
	Message string `json:"message"`
}

// LogPayload is the array wrapper
type LogPayload struct {
	Logs []LogMessage `json:"logs"`
}

type LogSearchHook struct {
	Endpoint   string
	Token      string
	HttpClient *http.Client
}

func NewLogSearchHook(endpoint string, token string) *LogSearchHook {
	return &LogSearchHook{
		Endpoint:   endpoint,
		Token:      token,
		HttpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (hook *LogSearchHook) Fire(entry *logrus.Entry) error {
	msg, err := entry.String()
	if err != nil {
		msg = entry.Message
	}

	payload := LogPayload{
		Logs: []LogMessage{
			{Message: msg},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil
	}

	req, err := http.NewRequest("POST", hook.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+hook.Token)

	// Fire and forget (in a real system, you'd batch and retry)
	go func() {
		hook.HttpClient.Do(req)
	}()

	return nil
}

func (hook *LogSearchHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
