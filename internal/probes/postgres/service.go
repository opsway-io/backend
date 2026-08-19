package postgres

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
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

	db, err := sql.Open("postgres", target)
	if err == nil {
		defer db.Close()
		
		ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		err = db.PingContext(ctxWithTimeout)
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
