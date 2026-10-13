package cmd

import (
	"errors"
	xhttp "net/http"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
)

func TestFormatAssertionFailureDescription_StatusCode(t *testing.T) {
	assertion := entities.MonitorAssertion{
		Source:   "STATUS_CODE",
		Operator: "EQUAL",
		Target:   "200",
	}

	res := &http.Result{
		Response: http.Response{
			StatusCode: 502,
			Header:     xhttp.Header{"Content-Type": []string{"text/html"}},
			Body:       []byte("<html>502 Bad Gateway</html>"),
		},
		Timing: http.Timing{
			Phases: http.TimingPhases{
				Total: 300 * time.Millisecond,
			},
		},
	}

	desc := formatAssertionFailureDescription(assertion, res)
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "Assertion Failed: STATUS_CODE EQUAL 200")
	assert.Contains(t, desc, "• Expected Target:  200")
	assert.Contains(t, desc, "• Actual Received:  502 (Bad Gateway)")
	assert.Contains(t, desc, "Bad Gateway: Upstream server or reverse proxy is unreachable")
	assert.Contains(t, desc, "=== Recommended Troubleshooting ===")
	assert.Contains(t, desc, "=== Diagnostic Failure Snapshot ===")
	assert.Contains(t, desc, "HTTP Status: 502")
}

func TestFormatAssertionFailureDescription_ResponseTime(t *testing.T) {
	assertion := entities.MonitorAssertion{
		Source:   "RESPONSE_TIME",
		Operator: "LESS_THAN",
		Target:   "500",
	}

	res := &http.Result{
		Response: http.Response{
			StatusCode: 200,
		},
		Timing: http.Timing{
			Phases: http.TimingPhases{
				DNSLookup:        10 * time.Millisecond,
				TCPConnection:    20 * time.Millisecond,
				TLSHandshake:     30 * time.Millisecond,
				ServerProcessing: 1200 * time.Millisecond,
				ContentTransfer:  10 * time.Millisecond,
				Total:            1270 * time.Millisecond,
			},
		},
	}

	desc := formatAssertionFailureDescription(assertion, res)
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "Assertion Failed: RESPONSE_TIME LESS_THAN 500")
	assert.Contains(t, desc, "• Expected Target:  500")
	assert.Contains(t, desc, "• Actual Received:  1270ms")
	assert.Contains(t, desc, "Server Processing (TTFB) bottleneck")
	assert.Contains(t, desc, "Backend compute, database queries, or downstream API calls were slow")
}

func TestFormatAssertionFailureDescription_Headers(t *testing.T) {
	// 1. Missing header
	assertionMissing := entities.MonitorAssertion{
		Source:   "HEADERS",
		Property: "X-Custom-Header",
		Operator: "EQUAL",
		Target:   "expected-val",
	}

	res := &http.Result{
		Response: http.Response{
			StatusCode: 200,
			Header:     xhttp.Header{"Content-Type": []string{"application/json"}},
		},
	}

	descMissing := formatAssertionFailureDescription(assertionMissing, res)
	assert.Contains(t, descMissing, "• Property / Path:  X-Custom-Header")
	assert.Contains(t, descMissing, "• Actual Received:  <header not present in response>")
	assert.Contains(t, descMissing, "Header 'X-Custom-Header' was omitted by the server")

	// 2. Header with wrong value
	assertionWrong := entities.MonitorAssertion{
		Source:   "HEADERS",
		Property: "Content-Type",
		Operator: "CONTAINS",
		Target:   "text/xml",
	}

	descWrong := formatAssertionFailureDescription(assertionWrong, res)
	assert.Contains(t, descWrong, "• Actual Received:  application/json")
	assert.Contains(t, descWrong, "failing assertion 'CONTAINS text/xml'")
}

func TestFormatAssertionFailureDescription_JSONBody(t *testing.T) {
	assertion := entities.MonitorAssertion{
		Source:   "JSON_BODY",
		Property: "status",
		Operator: "EQUAL",
		Target:   "ok",
	}

	res := &http.Result{
		Response: http.Response{
			StatusCode: 200,
			Body:       []byte(`{"status": "degraded", "code": 503}`),
		},
	}

	desc := formatAssertionFailureDescription(assertion, res)
	assert.Contains(t, desc, "• Property / Path:  status")
	assert.Contains(t, desc, "• Actual Received:  degraded")
	assert.Contains(t, desc, "does not satisfy 'EQUAL ok'")

	// Missing path
	assertionMissingPath := entities.MonitorAssertion{
		Source:   "JSON_BODY",
		Property: "nonexistent.key",
		Operator: "NOT_EMPTY",
	}
	descMissingPath := formatAssertionFailureDescription(assertionMissingPath, res)
	assert.Contains(t, descMissingPath, "<path 'nonexistent.key' not found>")
}

func TestFormatTargetDownDescription(t *testing.T) {
	mon := &entities.Monitor{
		Name: "Production Auth Gateway",
		Steps: []entities.MonitorStep{
			{Method: "POST", URL: "https://auth.example.com/oauth/token"},
		},
	}

	netErr := errors.New("dial tcp 192.0.2.1:443: connect: connection refused")
	desc := formatTargetDownDescription(mon, 3, nil, netErr)

	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, `Target service "Production Auth Gateway" is DOWN and unreachable.`)
	assert.Contains(t, desc, "POST https://auth.example.com/oauth/token")
	assert.Contains(t, desc, "Consecutive Failures:     3")
	assert.Contains(t, desc, "connect: connection refused")
	assert.Contains(t, desc, "Port Closed / Process Down: Host responded with TCP RST")
	assert.Contains(t, desc, "=== Recommended Troubleshooting ===")
}

func TestFormatAuthFailedDescription(t *testing.T) {
	mon := &entities.Monitor{
		Name: "Partner API",
		Settings: entities.MonitorSettings{
			Auth: entities.MonitorSettingsAuth{
				Method:   "OAUTH2_CLIENT_CREDENTIALS",
				TokenURL: "https://auth.partner.com/oauth/v2/token",
				ClientID: "client_998877",
			},
		},
	}

	authErr := errors.New("oauth2: cannot fetch token: 401 Unauthorized (invalid_client)")
	desc := formatAuthFailedDescription(mon, nil, authErr)

	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, `Authentication failed for monitor "Partner API".`)
	assert.Contains(t, desc, "Auth Method:   OAUTH2_CLIENT_CREDENTIALS")
	assert.Contains(t, desc, "Token URL:     https://auth.partner.com/oauth/v2/token")
	assert.Contains(t, desc, "Client ID:     client_998877")
	assert.Contains(t, desc, "Invalid Client Credentials: The client_id or client_secret was rejected")
}

func TestFormatAnomalyDescription(t *testing.T) {
	mon := &entities.Monitor{
		Name: "Checkout Service",
	}

	res := &http.Result{
		Timing: http.Timing{
			Phases: http.TimingPhases{
				DNSLookup:        5 * time.Millisecond,
				TCPConnection:    15 * time.Millisecond,
				TLSHandshake:     25 * time.Millisecond,
				ServerProcessing: 950 * time.Millisecond,
				ContentTransfer:  5 * time.Millisecond,
				Total:            1000 * time.Millisecond,
			},
		},
	}

	anomalyResp := &ForecasterPredictResponse{
		Anomalies:   []bool{true},
		Predictions: []float64{200.0},
		UpperBounds: []float64{450.0},
	}

	desc := formatAnomalyDescription(mon, res, anomalyResp)
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, `Latency anomaly detected for monitor "Checkout Service"!`)
	assert.Contains(t, desc, "Expected Baseline Latency: 200 ms")
	assert.Contains(t, desc, "Anomaly Threshold (Upper): 450 ms")
	assert.Contains(t, desc, "Observed Response Time:    1000 ms (+400.0% above baseline)")
	assert.Contains(t, desc, "Server Processing (TTFB) accounted for 95.0% of overall latency")
}

func TestFormatTLSDescriptions(t *testing.T) {
	resDeprecated := &http.Result{
		TLS: &http.TLS{
			Version: "TLS 1.0",
			Cipher:  "TLS_RSA_WITH_AES_128_CBC_SHA",
			Certificate: http.Certificate{
				Subject:   http.CertificateSubject{CommonName: "legacy.example.com"},
				Issuer:    http.CertificateIssuer{Organization: "Example CA"},
				NotBefore: time.Now().Add(-30 * 24 * time.Hour),
				NotAfter:  time.Now().Add(60 * 24 * time.Hour),
			},
		},
	}

	descDep := formatDeprecatedTLSDescription(resDeprecated)
	assert.Contains(t, descDep, "negotiated insecure TLS 1.0 (RFC 8996 Deprecated)")
	assert.Contains(t, descDep, "POODLE, BEAST")

	// Cert expiring in 7 days
	resExpiring := &http.Result{
		TLS: &http.TLS{
			Version: "TLS 1.3",
			Certificate: http.Certificate{
				Subject:   http.CertificateSubject{CommonName: "app.example.com"},
				Issuer:    http.CertificateIssuer{Organization: "Let's Encrypt"},
				NotBefore: time.Now().Add(-83 * 24 * time.Hour),
				NotAfter:  time.Now().Add(7 * 24 * time.Hour),
			},
		},
	}
	descExp := formatCertExpiryDescription(resExpiring, false, 7)
	assert.Contains(t, descExp, `WARNING: SSL/TLS certificate for "app.example.com" will expire in 7 days`)
	assert.Contains(t, descExp, "Trigger certificate renewal via automated ACME client")
}
