package redis

import (
	"context"
	"time"

	probeHttp "github.com/opsway-io/backend/internal/probes/http"
	"github.com/redis/go-redis/v9"
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

	opt, err := redis.ParseURL(target)
	if err == nil {
		opt.DialTimeout = timeout
		opt.ReadTimeout = timeout

		client := redis.NewClient(opt)
		defer client.Close()

		ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		err = client.Ping(ctxWithTimeout).Err()
	}

	duration := time.Since(start)

	res := &probeHttp.Result{
		Timing: probeHttp.Timing{
			Phases: probeHttp.TimingPhases{
				TCPConnection: duration,
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
	}

	return res, nil
}
