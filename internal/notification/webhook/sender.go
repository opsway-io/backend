package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

const DefaultTimeout = 5 * time.Second

// Sender posts JSON payloads to webhook URLs.
type Sender interface {
	Post(ctx context.Context, url string, payload any) error
}

type HTTPSender struct {
	client *http.Client
}

func NewSender(timeout time.Duration) Sender {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	return &HTTPSender{
		client: &http.Client{Timeout: timeout},
	}
}

func (s *HTTPSender) Post(ctx context.Context, url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal webhook payload")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return errors.Wrap(err, "failed to create webhook request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "opsway-alerting/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send webhook")
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status code %d", resp.StatusCode)
	}

	return nil
}

// ErrDisallowedAddress is returned when a public webhook resolves to a non-public IP address.
var ErrDisallowedAddress = errors.New("webhook address is not allowed")

// NewPublicSender returns a sender for URLs supplied by untrusted (anonymous) users. It only allows
// http(s) and refuses to connect to loopback, private, link-local and other non-public addresses.
func NewPublicSender(timeout time.Duration) Sender {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	dialer := &net.Dialer{
		Timeout: timeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}

			ip := net.ParseIP(host)
			if ip == nil || !IsPublicIP(ip) {
				return ErrDisallowedAddress
			}

			return nil
		},
	}

	return &HTTPSender{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy:               nil,
				DialContext:         dialer.DialContext,
				TLSHandshakeTimeout: timeout,
			},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// IsPublicIP reports whether ip is a globally routable unicast address.
func IsPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return false
	}

	// Carrier-grade NAT 100.64.0.0/10
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1]&0xc0 == 64 {
		return false
	}

	return true
}
