package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	probeHttp "github.com/opsway-io/backend/internal/probes/http"
)

type Service interface {
	Probe(ctx context.Context, target string, timeout time.Duration) (*probeHttp.Result, error)
}

type ServiceImpl struct{}

func NewService() Service {
	return &ServiceImpl{}
}

// urlToDSN converts mysql://user:pass@host:port/dbname to user:pass@tcp(host:port)/dbname
func urlToDSN(target string) string {
	if !strings.HasPrefix(target, "mysql://") {
		return target
	}
	u, err := url.Parse(target)
	if err != nil {
		return target
	}

	dsn := ""
	if u.User != nil {
		dsn += u.User.String() + "@"
	}
	
	host := u.Host
	if host != "" {
		dsn += fmt.Sprintf("tcp(%s)", host)
	}

	if u.Path != "" {
		dsn += u.Path
	} else {
		dsn += "/"
	}

	if u.RawQuery != "" {
		dsn += "?" + u.RawQuery
	}

	return dsn
}

func (s *ServiceImpl) Probe(ctx context.Context, target string, timeout time.Duration) (*probeHttp.Result, error) {
	start := time.Now()

	dsn := urlToDSN(target)

	db, err := sql.Open("mysql", dsn)
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
