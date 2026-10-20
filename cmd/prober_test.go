package cmd

import (
	"context"
	"net"
	xhttp "net/http"
	"strings"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	incidentMocks "github.com/opsway-io/backend/internal/incident/mocks"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBuildDiagnosticSnapshot(t *testing.T) {
	headers := make(xhttp.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Custom-Trace", "trace-12345")

	res := &http.Result{
		Response: http.Response{
			StatusCode: 502,
			Header:     headers,
			Body:       []byte(`{"error": "bad gateway", "details": "upstream service timed out"}`),
		},
		Timing: http.Timing{
			Phases: http.TimingPhases{
				DNSLookup:        15 * time.Millisecond,
				TCPConnection:    25 * time.Millisecond,
				TLSHandshake:     40 * time.Millisecond,
				ServerProcessing: 1200 * time.Millisecond,
				ContentTransfer:  5 * time.Millisecond,
				Total:            1285 * time.Millisecond,
			},
		},
	}

	snapshot := BuildDiagnosticSnapshot(res)
	assert.NotEmpty(t, snapshot)
	assert.Contains(t, snapshot, "=== Diagnostic Failure Snapshot ===")
	assert.Contains(t, snapshot, "HTTP Status: 502")
	assert.Contains(t, snapshot, "DNS Lookup: 15ms")
	assert.Contains(t, snapshot, "Content-Type: application/json")
	assert.Contains(t, snapshot, "X-Custom-Trace: trace-12345")
	assert.Contains(t, snapshot, `bad gateway`)

	// Nil result returns empty string safely
	assert.Empty(t, BuildDiagnosticSnapshot(nil))
}

func TestDiagnosticSnapshotTruncation(t *testing.T) {
	longBody := strings.Repeat("A", 10000)
	res := &http.Result{
		Response: http.Response{
			StatusCode: 500,
			Body:       []byte(longBody),
		},
	}

	snapshot := BuildDiagnosticSnapshot(res)
	assert.Contains(t, snapshot, "[truncated 4KB preview]")
	assert.Less(t, len(snapshot), 5500)
}

func TestVariableChainingAndExtraction(t *testing.T) {
	extracted := make(map[string]string)
	extracted["token"] = "eyJhGciOi..."
	extracted["userId"] = "42"

	// Chaining in URL
	urlPattern := "https://api.example.com/users/{{userId}}/profile"
	for k, v := range extracted {
		urlPattern = strings.ReplaceAll(urlPattern, "{{"+k+"}}", v)
	}
	assert.Equal(t, "https://api.example.com/users/42/profile", urlPattern)

	// Chaining in Header
	headerPattern := "Bearer {{token}}"
	for k, v := range extracted {
		headerPattern = strings.ReplaceAll(headerPattern, "{{"+k+"}}", v)
	}
	assert.Equal(t, "Bearer eyJhGciOi...", headerPattern)

	// Chaining in Body
	bodyPattern := `{"user_id": {{userId}}, "auth": "{{token}}"}`
	for k, v := range extracted {
		bodyPattern = strings.ReplaceAll(bodyPattern, "{{"+k+"}}", v)
	}
	assert.Equal(t, `{"user_id": 42, "auth": "eyJhGciOi..."}`, bodyPattern)
}

func TestCheckAnomaly_Disabled(t *testing.T) {
	res := &http.Result{}
	cfg := ForecasterConfig{
		Enabled: false,
		URL:     "http://localhost:9999/predict",
	}

	resp, err := checkAnomaly(context.Background(), cfg, 1, res)
	assert.NoError(t, err)
	assert.Nil(t, resp)
}

func TestCheckAnomaly_Timeout(t *testing.T) {
	// Spin up a slow server that takes longer than 1s timeout
	slowServer := xhttp.HandlerFunc(func(w xhttp.ResponseWriter, r *xhttp.Request) {
		time.Sleep(1500 * time.Millisecond)
		w.WriteHeader(xhttp.StatusOK)
	})
	s := &xhttp.Server{
		Addr:    "127.0.0.1:0",
		Handler: slowServer,
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer ln.Close()

	go func() {
		_ = s.Serve(ln)
	}()
	defer s.Close()

	res := &http.Result{}
	cfg := ForecasterConfig{
		Enabled: true,
		URL:     "http://" + ln.Addr().String() + "/predict",
	}

	start := time.Now()
	resp, err := checkAnomaly(context.Background(), cfg, 1, res)
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Less(t, elapsed, 1300*time.Millisecond)
}

func TestTriggerIncident_UsesMonitorIDWhenAssertionMonitorIDIsZero(t *testing.T) {
	mockIncidentSvc := incidentMocks.NewService(t)
	m := &entities.Monitor{
		ID:     42,
		TeamID: 7,
		Name:   "API Monitor",
	}
	res := &http.Result{
		Response: http.Response{StatusCode: 500},
	}
	failed := []entities.MonitorAssertion{
		{
			ID:        101,
			MonitorID: 0, // Unpopulated on multi-step assertion
			Source:    "STATUS_CODE",
			Operator:  "EQUAL",
			Target:    "200",
		},
	}

	mockIncidentSvc.On("Upsert", mock.Anything, mock.MatchedBy(func(incidents *[]entities.Incident) bool {
		if incidents == nil || len(*incidents) != 1 {
			return false
		}
		inc := (*incidents)[0]
		return inc.MonitorID != nil && *inc.MonitorID == 42 && inc.TeamID == 7 && inc.MonitorAssertionID != nil && *inc.MonitorAssertionID == 101
	})).Return(nil)
	mockIncidentSvc.On("CreateOccurrence", mock.Anything, mock.Anything).Return(nil)

	err := triggerIncident(context.Background(), m, res, &failed, mockIncidentSvc)
	assert.NoError(t, err)
}
