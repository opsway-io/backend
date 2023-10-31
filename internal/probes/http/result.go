package http

import (
	"net/http"
	"time"
)

type Result struct {
	Response Response
	Timing   Timing
	TLS      *TLS
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

type Timing struct {
	Phases TimingPhases
}

type TimingPhases struct {
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	ContentTransfer  time.Duration
	Total            time.Duration
}
type TLS struct {
	Version     string
	Cipher      string
	Certificate Certificate
}

type Certificate struct {
	Issuer     CertificateIssuer
	Subject    CertificateSubject
	NotBefore  time.Time
	NotAfter   time.Time
	NotExpired bool
	HostValid  bool
	TrustedCA  bool
}

type CertificateSubject struct {
	CommonName string
}

type CertificateIssuer struct {
	Organization string
}
