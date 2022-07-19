package http

import "time"

type Result struct {
	Response Response
	Timing   Timing
	SSL      *SSL
}

type Response struct {
	StatusCode int
}

type Timing struct {
	Phases   TimingPhases
	Timeline TimingTimeline
}

type TimingPhases struct {
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	ContentTransfer  time.Duration
}

type TimingTimeline struct {
	NameLookup    time.Duration
	Connect       time.Duration
	PreTransfer   time.Duration
	StartTransfer time.Duration
	Total         time.Duration
}

type SSL struct {
	Version     string
	Cipher      string
	Certificate Certificate
}

type Certificate struct {
	Issuer    CertificateIssuer
	Subject   CertificateSubject
	NotBefore time.Time
	NotAfter  time.Time
}

type CertificateSubject struct {
	CommonName string
}

type CertificateIssuer struct {
	CommonName   string
	Organization []string
	Country      []string
}
