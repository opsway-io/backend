package websocket

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	probeHttp "github.com/opsway-io/backend/internal/probes/http"
)

type Service interface {
	Probe(ctx context.Context, target string, timeout time.Duration) (*probeHttp.Result, error)
}

type ServiceImpl struct{}

func NewService() Service {
	return &ServiceImpl{}
}

func (s *ServiceImpl) Probe(ctx context.Context, target string, timeout time.Duration) (*probeHttp.Result, error) {
	start := time.Now()

	dialer := websocket.Dialer{
		HandshakeTimeout: timeout,
	}

	conn, _, err := dialer.DialContext(ctx, target, nil)

	duration := time.Since(start)

	res := &probeHttp.Result{
		Timing: probeHttp.Timing{
			Phases: probeHttp.TimingPhases{
				TCPConnection: duration,
				Total:         duration,
			},
		},
		Response: probeHttp.Response{
			StatusCode: 101, // Switching Protocols
			Body:       []byte("OK"),
		},
	}

	if err != nil {
		res.Response.StatusCode = 503
		res.Response.Body = []byte(err.Error())
	} else {
		conn.Close()
	}

	return res, nil
}
