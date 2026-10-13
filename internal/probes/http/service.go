package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
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
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Set user agent
	req.Header.Set("User-Agent", s.config.UserAgent)

	// Instrument the request with httpstat
	var result httpstat.Result
	httpStatCtx := httpstat.WithHTTPStat(ctx, &result)
	req = req.WithContext(httpStatCtx)

	type tlsHop struct {
		hostname string
		tlsState *tls.ConnectionState
	}
	var tlsHops []tlsHop

	client := s.newHttpClient(timeout)
	client.CheckRedirect = func(req *xhttp.Request, via []*xhttp.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		if req.Response != nil && req.Response.TLS != nil && len(req.Response.TLS.PeerCertificates) > 0 {
			prevReq := via[len(via)-1]
			tlsHops = append(tlsHops, tlsHop{
				hostname: prevReq.URL.Hostname(),
				tlsState: req.Response.TLS,
			})
		}
		return nil
	}

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
	_ = resp.Body.Close()

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
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		finalReq := resp.Request
		finalHostname := req.URL.Hostname()
		if finalReq != nil && finalReq.URL != nil {
			finalHostname = finalReq.URL.Hostname()
		}
		tlsHops = append(tlsHops, tlsHop{
			hostname: finalHostname,
			tlsState: resp.TLS,
		})
	}

	if len(tlsHops) > 0 {
		type evaluatedHop struct {
			hop        tlsHop
			cert       *x509.Certificate
			notExpired bool
			hostValid  bool
			trustedCA  bool
		}

		evaluated := make([]evaluatedHop, len(tlsHops))
		var firstFailing *evaluatedHop

		for i, h := range tlsHops {
			cert := h.tlsState.PeerCertificates[0]
			ne := s.certificateNotExpired(cert)
			hv := s.certificateHostValid(cert, h.hostname)
			tca := s.certificateTrusted(h.tlsState, h.hostname)

			eh := evaluatedHop{
				hop:        h,
				cert:       cert,
				notExpired: ne,
				hostValid:  hv,
				trustedCA:  tca,
			}
			evaluated[i] = eh

			if firstFailing == nil && (!hv || !tca || !ne) {
				firstFailing = &evaluated[i]
			}
		}

		chosen := evaluated[0]
		if firstFailing != nil {
			chosen = *firstFailing
		} else {
			// If all hops are valid, prefer the hop with the earliest expiring certificate
			for _, eh := range evaluated[1:] {
				if eh.cert.NotAfter.Before(chosen.cert.NotAfter) {
					chosen = eh
				}
			}
		}

		meta.TLS = &TLS{
			Version: TLSVersionName(chosen.hop.tlsState.Version),
			Cipher:  tls.CipherSuiteName(chosen.hop.tlsState.CipherSuite),
			Certificate: Certificate{
				Issuer: CertificateIssuer{
					Organization: strings.Join(chosen.cert.Issuer.Organization, ""),
				},
				Subject: CertificateSubject{
					CommonName: chosen.cert.Subject.CommonName,
				},
				NotBefore:  chosen.cert.NotBefore,
				NotAfter:   chosen.cert.NotAfter,
				NotExpired: chosen.notExpired,
				HostValid:  chosen.hostValid,
				TrustedCA:  chosen.trustedCA,
			},
		}
	} else if resp.TLS != nil {
		meta.TLS = &TLS{
			Version: TLSVersionName(resp.TLS.Version),
			Cipher:  tls.CipherSuiteName(resp.TLS.CipherSuite),
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
		Transport: &xhttp.Transport{
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

func (s *ServiceImpl) certificateTrusted(state *tls.ConnectionState, host string) bool {
	opts := x509.VerifyOptions{
		DNSName:       host,
		Intermediates: x509.NewCertPool(),
	}
	for _, cert := range state.PeerCertificates[1:] {
		opts.Intermediates.AddCert(cert)
	}
	_, err := state.PeerCertificates[0].Verify(opts)
	return err == nil
}
