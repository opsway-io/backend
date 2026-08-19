package dns

import (
	"context"
	"net"
	"strings"
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

    recordType := "A"
    host := target
    
    // Simple parsing if format is host?type=MX
    if strings.Contains(target, "?type=") {
        parts := strings.Split(target, "?type=")
        host = parts[0]
        recordType = strings.ToUpper(parts[1])
    }

	resolver := net.Resolver{}
    var err error

    switch recordType {
    case "MX":
        _, err = resolver.LookupMX(timeoutCtx, host)
    case "TXT":
        _, err = resolver.LookupTXT(timeoutCtx, host)
    case "CNAME":
        _, err = resolver.LookupCNAME(timeoutCtx, host)
    case "AAAA":
        // go net package handles both IPv4 and IPv6 for LookupIP
        var ips []net.IPAddr
        ips, err = resolver.LookupIPAddr(timeoutCtx, host)
        hasAAAA := false
        for _, ip := range ips {
            if ip.IP.To4() == nil {
                hasAAAA = true
                break
            }
        }
        if err == nil && !hasAAAA {
            err = &net.DNSError{Err: "no AAAA records found", Name: host}
        }
    default:
        _, err = resolver.LookupIPAddr(timeoutCtx, host)
    }
	
	duration := time.Since(start)

	res := &probeHttp.Result{
		Timing: probeHttp.Timing{
			Phases: probeHttp.TimingPhases{
				DNSLookup: duration,
				Total:     duration,
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
