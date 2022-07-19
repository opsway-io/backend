package http

import btls "crypto/tls"

//nolint:gochecknoglobals
var versions = map[uint16]string{
	btls.VersionSSL30: "SSL",
	btls.VersionTLS10: "TLS 1.0",
	btls.VersionTLS11: "TLS 1.1",
	btls.VersionTLS12: "TLS 1.2",
	btls.VersionTLS13: "TLS 1.3",
}

func TLSVersionName(version uint16) string {
	return versions[version]
}
