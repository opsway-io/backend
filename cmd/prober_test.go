package cmd

import (
	xhttp "net/http"
	"strings"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
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
