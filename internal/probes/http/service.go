package http

import (
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	xhttp "net/http"
	"strings"
	"time"

	"github.com/opsway-io/go-httpstat"
	"github.com/pkg/errors"
)

type Config struct {
	UserAgent   string        `mapstructure:"user_agent" default:"opsway 1.0.0"`
	DNSTimeout  time.Duration `mapstructure:"dns_timeout" default:"5s"`
	DNSPort     int           `mapstructure:"dns_port" default:"53"`
	DNSAddress  string        `mapstructure:"dns_address" default:"8.8.8.8"`
	DNSProtocol string        `mapstructure:"dns_protocol" default:"udp"`
}

type Service interface {
	Probe(ctx context.Context, method, url string, headers map[string]string, body io.Reader, timeout time.Duration) (*Result, error)
}

type ServiceImpl struct {
	config Config
}

func NewService(config Config) Service {
	return &ServiceImpl{
		config: config,
	}
}

func (s *ServiceImpl) Probe(ctx context.Context, method, url string, headers map[string]string, body io.Reader, timeout time.Duration) (*Result, error) {
	req, err := xhttp.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	req.Header.Set("User-Agent", s.config.UserAgent)

	var result httpstat.Result
	httpStatCtx := httpstat.WithHTTPStat(ctx, &result)
	req = req.WithContext(httpStatCtx)

	client := s.newHttpClient(timeout)

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}

	if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	resp.Body.Close()
	result.End(time.Now())

	meta := &Result{
		Response: Response{
			StatusCode: resp.StatusCode,
		},
		Timing: Timing{
			Phases: TimingPhases{
				DNSLookup:        result.DNSLookup,
				TCPConnection:    result.TCPConnection,
				TLSHandshake:     result.TLSHandshake,
				ServerProcessing: result.ServerProcessing,
				ContentTransfer:  result.ContentTransfer,
				Total:            result.Total,
			},
		},
	}

	if resp.TLS != nil {
		meta.TLS = &TLS{
			Version: TLSVersionName(resp.TLS.Version),
			Cipher:  tls.CipherSuiteName(resp.TLS.CipherSuite),
		}

		if resp.TLS.PeerCertificates != nil && len(resp.TLS.PeerCertificates) > 0 {
			cert := resp.TLS.PeerCertificates[0]

			meta.TLS.Certificate = Certificate{
				Issuer: CertificateIssuer{
					Organization: strings.Join(cert.Issuer.Organization, ""),
				},
				Subject: CertificateSubject{
					CommonName: cert.Subject.CommonName,
				},
				NotBefore: cert.NotBefore,
				NotAfter:  cert.NotAfter,
			}
		}
	}

	return meta, nil
}

func (s *ServiceImpl) newHttpClient(timeout time.Duration) *xhttp.Client {
	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: s.config.DNSTimeout,
				}

				return d.DialContext(ctx, s.config.DNSProtocol, s.config.DNSAddress)
			},
		},
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}

	return &xhttp.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: dialContext,
		},
	}
}
