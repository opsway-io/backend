package http

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	xhttp "net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/opsway-io/go-httpstat"
)

const (
	UserAgent = "opsway 1.0.0"
)

func Probe(method, url string, headers map[string]string, body io.Reader, timeout time.Duration) (*Result, error) {
	req, err := xhttp.NewRequest(method, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	req.Header.Set("User-Agent", UserAgent)

	var result httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &result)
	req = req.WithContext(ctx)

	client := &xhttp.Client{
		Timeout: timeout,
	}

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
