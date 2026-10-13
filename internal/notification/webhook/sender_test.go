package webhook

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPublicIP(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.1.2.3", "192.168.0.1", "172.16.5.4", "169.254.169.254", "::1", "fd00::1", "100.64.1.1", "0.0.0.0"} {
		assert.False(t, IsPublicIP(net.ParseIP(ip)), ip)
	}

	for _, ip := range []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"} {
		assert.True(t, IsPublicIP(net.ParseIP(ip)), ip)
	}
}

func TestPublicSenderRefusesLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()

	err := NewPublicSender(0).Post(context.Background(), srv.URL, map[string]string{"a": "b"})
	assert.ErrorIs(t, err, ErrDisallowedAddress)

	assert.NoError(t, NewSender(0).Post(context.Background(), srv.URL, map[string]string{"a": "b"}))
}
