package tcp_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/probes/tcp"
	"github.com/stretchr/testify/assert"
)

func TestTCPProbe_Success(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err == nil {
			conn.Close()
		}
	}()

	svc := tcp.NewService()
	res, err := svc.Probe(context.Background(), ln.Addr().String(), 2*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 200, res.Response.StatusCode)
	assert.Equal(t, []byte("OK"), res.Response.Body)
}

func TestTCPProbe_Failure(t *testing.T) {
	svc := tcp.NewService()
	// Target closed/unbound local port
	res, err := svc.Probe(context.Background(), "127.0.0.1:59999", 500*time.Millisecond)
	assert.Error(t, err)
	assert.Nil(t, res)
}
