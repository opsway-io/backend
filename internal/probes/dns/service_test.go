package dns_test

import (
	"context"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/probes/dns"
	"github.com/stretchr/testify/assert"
)

func TestDNSProbe_Success(t *testing.T) {
	svc := dns.NewService()
	res, err := svc.Probe(context.Background(), "localhost", 2*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 200, res.Response.StatusCode)
}

func TestDNSProbe_Failure(t *testing.T) {
	svc := dns.NewService()
	res, err := svc.Probe(context.Background(), "nonexistent-domain-that-does-not-exist.invalid", 1*time.Second)
	assert.Error(t, err)
	assert.Nil(t, res)
}
