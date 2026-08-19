package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	probeHttp "github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPProbeService(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "opsway 1.0.0", r.Header.Get("User-Agent"))
		if r.Header.Get("X-Custom") != "" {
			w.Header().Set("X-Received", "true")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Default config
	cfg := probeHttp.Config{
		UserAgent:            "opsway 1.0.0",
		DNSTimeout:           5 * time.Second,
		MaxBodyBytesReadSize: 1048576,
	}

	svc := probeHttp.NewService(cfg)
	ctx := context.Background()

	t.Run("successful http probe", func(t *testing.T) {
		headers := map[string]string{
			"X-Custom": "test",
		}
		res, err := svc.Probe(ctx, "GET", server.URL, headers, nil, 2*time.Second)

		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 200, res.Response.StatusCode)
		assert.Equal(t, []byte("OK"), res.Response.Body)
		assert.Equal(t, "true", res.Response.Header.Get("X-Received"))
		assert.Greater(t, res.Timing.Phases.Total, time.Duration(0))
		assert.Nil(t, res.TLS) // No TLS for http://
	})
}

func TestTLSProbeService(t *testing.T) {
	// Create a mock TLS server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Secure OK"))
	}))
	defer server.Close()

	cfg := probeHttp.Config{
		UserAgent:            "opsway 1.0.0",
		DNSTimeout:           5 * time.Second,
		MaxBodyBytesReadSize: 1048576,
	}

	svc := probeHttp.NewService(cfg)
	ctx := context.Background()

	t.Run("successful https probe with TLS checks", func(t *testing.T) {
		res, err := svc.Probe(ctx, "GET", server.URL, nil, nil, 2*time.Second)

		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 200, res.Response.StatusCode)
		assert.NotNil(t, res.TLS)

		// Assert TLS fields
		assert.NotEmpty(t, res.TLS.Version)
		assert.NotEmpty(t, res.TLS.Cipher)
		// It's a self-signed cert from httptest, so TrustedCA should be false
		assert.False(t, res.TLS.Certificate.TrustedCA)
		// Should be not expired
		assert.True(t, res.TLS.Certificate.NotExpired)
	})
}
