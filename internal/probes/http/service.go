package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	xhttp "net/http"
	"strings"
	"time"

	"github.com/opsway-io/go-httpstat"
	"github.com/pkg/errors"
)

type Config struct {
	UserAgent string `mapstructure:"user_agent" default:"opsway 1.0.0"`

	DNSTimeout  time.Duration `mapstructure:"dns_timeout" default:"5s"`
	DNSPort     int           `mapstructure:"dns_port" default:"53"`
	DNSAddress  string        `mapstructure:"dns_address" default:"8.8.8.8"`
	DNSProtocol string        `mapstructure:"dns_protocol" default:"udp"`

	// Max number of bytes to be read from response body, defaults to 1MB
	MaxBodyBytesReadSize int64 `mapstructure:"max_body_bytes_read_size" default:"1048576"`
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
	// Initialize the request
	req, err := xhttp.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	// Set headers
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	// Set user agent
	req.Header.Set("User-Agent", s.config.UserAgent)

	// Instrument the request with httpstat
	var result httpstat.Result
	httpStatCtx := httpstat.WithHTTPStat(ctx, &result)
	req = req.WithContext(httpStatCtx)

	client := s.newHttpClient(timeout)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}

	// Read the response body
	limitedReader := &io.LimitedReader{R: resp.Body, N: s.config.MaxBodyBytesReadSize}
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	// Close the response body to avoid leaking resources
	resp.Body.Close()

	// End the httpstat timer
	result.End(time.Now())

	// Create the result
	meta := &Result{
		Response: Response{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Body:       bodyBytes,
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

	// Add TLS information if available
	if resp.TLS != nil {
		meta.TLS = &TLS{
			Version: TLSVersionName(resp.TLS.Version),
			Cipher:  tls.CipherSuiteName(resp.TLS.CipherSuite),
		}

		if resp.TLS.PeerCertificates != nil && len(resp.TLS.PeerCertificates) > 0 {
			cert := resp.TLS.PeerCertificates[0]
			hostname := req.URL.Hostname()

			trustedCA := true // TODO: check if the CA is trusted

			notExpired := s.certificateNotExpired(cert)
			hostValid := s.certificateHostValid(cert, hostname)

			meta.TLS.Certificate = Certificate{
				Issuer: CertificateIssuer{
					Organization: strings.Join(cert.Issuer.Organization, ""),
				},
				Subject: CertificateSubject{
					CommonName: cert.Subject.CommonName,
				},
				NotBefore:  cert.NotBefore,
				NotAfter:   cert.NotAfter,
				NotExpired: notExpired,
				HostValid:  hostValid,
				TrustedCA:  trustedCA,
			}
		}
	}

	return meta, nil
}

func (s *ServiceImpl) newHttpClient(timeout time.Duration) *xhttp.Client {
	// Create a custom dialer to set a custom DNS resolver
	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: s.config.DNSTimeout,
				}

				return d.DialContext(ctx, s.config.DNSProtocol, fmt.Sprintf("%s:%d", s.config.DNSAddress, s.config.DNSPort))
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
			TLSClientConfig: &tls.Config{
				// We don't want to verify the certificate here
				// because we want to get the certificate information
				// even if it's expired or the host is invalid
				InsecureSkipVerify: true, // nolint:gosec
			},
		},
	}
}

func (s *ServiceImpl) certificateNotExpired(cert *x509.Certificate) (notExpired bool) {
	now := time.Now()

	return now.Before(cert.NotAfter) && now.After(cert.NotBefore)
}

func (s *ServiceImpl) certificateHostValid(cert *x509.Certificate, host string) (hostValid bool) {
	return cert.VerifyHostname(host) == nil
}
