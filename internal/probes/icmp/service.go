package icmp

import (
	"context"
	"os/exec"
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

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "ping", "-c", "1", target)
	err := cmd.Run()

	duration := time.Since(start)

	res := &probeHttp.Result{
		Timing: probeHttp.Timing{
			Phases: probeHttp.TimingPhases{
				Total: duration,
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
	}

	return res, nil
}
