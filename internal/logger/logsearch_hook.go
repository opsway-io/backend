package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// LogMessage matches the struct expected by log-search
type LogMessage struct {
	Message     string                 `json:"message"`
	Timestamp   string                 `json:"timestamp"`
	Level       string                 `json:"level"`
	ServiceName string                 `json:"service_name"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
}

// LogPayload is the array wrapper
type LogPayload struct {
	Logs []LogMessage `json:"logs"`
}

type LogSearchHook struct {
	Endpoint    string
	Token       string
	ServiceName string
	HttpClient  *http.Client
}

func detectServiceName() string {
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		return os.Args[1]
	}
	if svc := os.Getenv("OPSWAY_SERVICE_NAME"); svc != "" {
		return svc
	}
	return "backend"
}

func NewLogSearchHook(endpoint string, token string) *LogSearchHook {
	return &LogSearchHook{
		Endpoint:    endpoint,
		Token:       token,
		ServiceName: detectServiceName(),
		HttpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (hook *LogSearchHook) Fire(entry *logrus.Entry) error {
	if entry.Level == logrus.DebugLevel && entry.Message == "healthz check" {
		return nil
	}

	msg := entry.Message
	fields := make(map[string]interface{}, len(entry.Data))
	var kvParts []string
	for k, v := range entry.Data {
		fields[k] = fmt.Sprintf("%v", v)
		kvParts = append(kvParts, fmt.Sprintf("%s=%v", k, v))
	}
	if len(kvParts) > 0 {
		msg = fmt.Sprintf("%s (%s)", msg, strings.Join(kvParts, " "))
	}

	lvl := strings.ToUpper(entry.Level.String())
	if lvl == "WARNING" {
		lvl = "WARN"
	}

	payload := LogPayload{
		Logs: []LogMessage{
			{
				Message:     msg,
				Timestamp:   entry.Time.UTC().Format(time.RFC3339Nano),
				Level:       lvl,
				ServiceName: hook.ServiceName,
				Fields:      fields,
			},
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
		if resp, err := hook.HttpClient.Do(req); err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}()

	return nil
}

func (hook *LogSearchHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
