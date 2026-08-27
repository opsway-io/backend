package udp

import (
	"context"
	"net"
	"time"

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

	var d net.Dialer
	d.Timeout = timeout
	conn, err := d.DialContext(ctx, "udp", target)

	duration := time.Since(start)

	res := &probeHttp.Result{
		Timing: probeHttp.Timing{
			Phases: probeHttp.TimingPhases{
				TCPConnection: duration, // Map to TCPConnection phase for simplicity
				Total:         duration,
			},
		},
		Response: probeHttp.Response{
			StatusCode: 200,
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
