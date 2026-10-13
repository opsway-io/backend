package cmd

import (
	"encoding/json"
	"fmt"
	xhttp "net/http"
	"strings"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/yalp/jsonpath"
)

func extractActualAssertionValue(assertion entities.MonitorAssertion, hr *http.Result) (actual string, rootCauseHint string, checklist string) {
	if hr == nil {
		return "<no response received>", "The probe request failed to receive a response from the server.", "1. Verify host and port reachability.\n2. Check target application server status."
	}

	switch assertion.Source {
	case "STATUS_CODE":
		actual = fmt.Sprintf("%d (%s)", hr.Response.StatusCode, xhttp.StatusText(hr.Response.StatusCode))
		code := hr.Response.StatusCode
		switch {
		case code == 500:
			rootCauseHint = "Internal Server Error: Application crashed or threw an unhandled exception."
		case code == 502:
			rootCauseHint = "Bad Gateway: Upstream server or reverse proxy is unreachable, crashed, or returned an invalid response."
		case code == 503:
			rootCauseHint = "Service Unavailable: Upstream service is overloaded, failing health checks, or under maintenance."
		case code == 504:
			rootCauseHint = "Gateway Timeout: Upstream backend server took too long to respond to reverse proxy / ingress."
		case code == 401:
			rootCauseHint = "Unauthorized: Missing, expired, or invalid authentication credentials/token."
		case code == 403:
			rootCauseHint = "Forbidden: Client credentials lack required permissions or access was blocked by WAF/firewall."
		case code == 404:
			rootCauseHint = "Not Found: The requested endpoint URL or resource route was not found on the server."
		case code == 429:
			rootCauseHint = "Too Many Requests: Rate limit exceeded on target endpoint."
		case code >= 500:
			rootCauseHint = fmt.Sprintf("Server Error %d: The target server encountered an internal failure processing the request.", code)
		case code >= 400:
			rootCauseHint = fmt.Sprintf("Client Error %d: The server rejected the request due to client-side error or validation failure.", code)
		default:
			rootCauseHint = fmt.Sprintf("HTTP status code %d does not satisfy condition '%s %s'.", code, assertion.Operator, assertion.Target)
		}
		checklist = "1. Inspect application server error logs and stack traces around failure timestamp.\n2. Check downstream dependencies (database, Redis, third-party microservices).\n3. Verify HTTP request parameters, path, and authorization headers.\n4. Check server resource metrics (CPU, RAM, open file descriptors)."

	case "RESPONSE_TIME":
		actual = fmt.Sprintf("%dms", hr.Timing.Phases.Total.Milliseconds())
		total := hr.Timing.Phases.Total
		sp := hr.Timing.Phases.ServerProcessing
		dns := hr.Timing.Phases.DNSLookup
		tcp := hr.Timing.Phases.TCPConnection
		tls := hr.Timing.Phases.TLSHandshake
		transfer := hr.Timing.Phases.ContentTransfer

		if total > 0 && float64(sp)/float64(total) >= 0.5 {
			rootCauseHint = fmt.Sprintf("Server Processing (TTFB) bottleneck: %v (%.0f%% of total latency). Backend compute, database queries, or downstream API calls were slow.", sp, float64(sp)/float64(total)*100)
		} else if total > 0 && float64(dns)/float64(total) >= 0.4 {
			rootCauseHint = fmt.Sprintf("DNS Lookup bottleneck: %v (%.0f%% of total latency). DNS nameserver resolution latency or throttling.", dns, float64(dns)/float64(total)*100)
		} else if total > 0 && float64(tcp)/float64(total) >= 0.4 {
			rootCauseHint = fmt.Sprintf("TCP Connection bottleneck: %v (%.0f%% of total latency). Network congestion or high round-trip packet latency.", tcp, float64(tcp)/float64(total)*100)
		} else if total > 0 && float64(tls)/float64(total) >= 0.4 {
			rootCauseHint = fmt.Sprintf("TLS Handshake bottleneck: %v (%.0f%% of total latency). Sluggish cryptographic negotiation or packet loss.", tls, float64(tls)/float64(total)*100)
		} else if total > 0 && float64(transfer)/float64(total) >= 0.4 {
			rootCauseHint = fmt.Sprintf("Content Transfer bottleneck: %v (%.0f%% of total latency). Large payload size or constrained egress throughput.", transfer, float64(transfer)/float64(total)*100)
		} else {
			rootCauseHint = fmt.Sprintf("Observed response time (%v) exceeded target threshold '%s %s'.", total, assertion.Operator, assertion.Target)
		}
		checklist = "1. Analyze the largest phase contributor in the Timing Breakdown below.\n2. Check database query performance and connection pool saturation.\n3. Verify CPU and memory utilization on application hosts.\n4. Check network routing and latency to the host."

	case "HEADERS":
		vals := hr.Response.Header.Values(assertion.Property)
		if len(vals) == 0 {
			actual = "<header not present in response>"
			rootCauseHint = fmt.Sprintf("Header '%s' was omitted by the server or stripped by an intermediate reverse proxy / CDN.", assertion.Property)
		} else {
			actual = strings.Join(vals, ", ")
			rootCauseHint = fmt.Sprintf("Header '%s' had value '%s', failing assertion '%s %s'.", assertion.Property, actual, assertion.Operator, assertion.Target)
		}
		checklist = "1. Verify web server / reverse proxy configuration (e.g. Nginx, Cloudflare, Traefik headers).\n2. Check application middleware responsible for setting response headers.\n3. Review CORS, caching, and security header policies."

	case "JSON_BODY":
		if len(hr.Response.Body) == 0 {
			actual = "<empty response body>"
			rootCauseHint = "Response body was empty; cannot evaluate JSONPath query."
		} else {
			var parsed interface{}
			if err := json.Unmarshal(hr.Response.Body, &parsed); err != nil {
				actual = "<invalid JSON payload>"
				rootCauseHint = fmt.Sprintf("Response body could not be parsed as JSON: %v", err)
			} else {
			queryPath := assertion.Property
			if !strings.HasPrefix(queryPath, "$") {
				if strings.HasPrefix(queryPath, ".") {
					queryPath = "$" + queryPath
				} else {
					queryPath = "$." + queryPath
				}
			}
			val, err := jsonpath.Read(parsed, queryPath)
			if err != nil {
				actual = fmt.Sprintf("<path '%s' not found>", assertion.Property)
				rootCauseHint = fmt.Sprintf("JSONPath query '%s' could not be resolved in the JSON payload: %v", assertion.Property, err)
			} else {
				actual = fmt.Sprintf("%v", val)
				rootCauseHint = fmt.Sprintf("JSONPath '%s' resolved to '%v', which does not satisfy '%s %s'.", assertion.Property, val, assertion.Operator, assertion.Target)
			}
			}
		}
		checklist = "1. Verify API contract and schema changes on the target service.\n2. Inspect response payload preview below for error messages or schema mismatches.\n3. Validate query parameters and request payload sent by the probe."

	case "RAW_BODY":
		if len(hr.Response.Body) == 0 {
			actual = "<empty response body>"
			rootCauseHint = "Response body was empty."
		} else {
			bodyStr := string(hr.Response.Body)
			if len(bodyStr) > 200 {
				actual = bodyStr[:200] + "... [truncated]"
			} else {
				actual = bodyStr
			}
			rootCauseHint = fmt.Sprintf("Response body did not satisfy condition '%s %s'.", assertion.Operator, assertion.Target)
		}
		checklist = "1. Review raw response body preview below.\n2. Check server logs to see why the expected content was not produced.\n3. Check if an error page (e.g. default 502 HTML) was returned instead of expected content."

	case "TLS":
		if hr.TLS == nil {
			actual = "<no TLS handshake data>"
			rootCauseHint = "Connection was not established over TLS or TLS handshake did not complete."
		} else {
			switch assertion.Operator {
			case "EXPIRED", "NOT_EXPIRED":
				actual = fmt.Sprintf("Expired: %v (Validity: %s to %s)", !hr.TLS.Certificate.NotExpired, hr.TLS.Certificate.NotBefore.Format("2006-01-02"), hr.TLS.Certificate.NotAfter.Format("2006-01-02"))
			case "EXPIRES_LESS_THAN", "EXPIRES_GREATER_THAN":
				daysRemaining := int(time.Until(hr.TLS.Certificate.NotAfter).Hours() / 24)
				actual = fmt.Sprintf("%d days remaining (Expires: %s)", daysRemaining, hr.TLS.Certificate.NotAfter.Format("2006-01-02"))
			default:
				actual = fmt.Sprintf("Version: %s, Cipher: %s", hr.TLS.Version, hr.TLS.Cipher)
			}
			rootCauseHint = fmt.Sprintf("TLS certificate parameters do not satisfy condition '%s %s'.", assertion.Operator, assertion.Target)
		}
		checklist = "1. Check SSL/TLS certificate expiration dates and CA issuance status.\n2. Renew certificate via ACME client (Certbot, cert-manager) if expiring soon.\n3. Ensure the complete certificate chain (fullchain.pem) is properly installed on the server."

	default:
		actual = "<unknown source>"
		rootCauseHint = fmt.Sprintf("Assertion condition '%s %s %s' was not met.", assertion.Source, assertion.Operator, assertion.Target)
		checklist = "1. Inspect monitor assertion configuration and probe results."
	}

	return actual, rootCauseHint, checklist
}

func formatAssertionFailureDescription(assertion entities.MonitorAssertion, hr *http.Result) string {
	actual, rootCauseHint, checklist := extractActualAssertionValue(assertion, hr)

	var sb strings.Builder
	titleStr := fmt.Sprintf("%s %s %s", assertion.Source, assertion.Operator, assertion.Target)
	if assertion.Property != "" {
		titleStr = fmt.Sprintf("%s (%s) %s %s", assertion.Source, assertion.Property, assertion.Operator, assertion.Target)
	}

	sb.WriteString(fmt.Sprintf("Assertion Failed: %s\n\n", titleStr))
	sb.WriteString("=== Assertion Comparison ===\n")
	sb.WriteString(fmt.Sprintf("• Assertion Source: %s\n", assertion.Source))
	if assertion.Property != "" {
		sb.WriteString(fmt.Sprintf("• Property / Path:  %s\n", assertion.Property))
	}
	sb.WriteString(fmt.Sprintf("• Operator:         %s\n", assertion.Operator))
	sb.WriteString(fmt.Sprintf("• Expected Target:  %s\n", assertion.Target))
	sb.WriteString(fmt.Sprintf("• Actual Received:  %s\n", actual))

	sb.WriteString("\n=== Root-Cause Diagnosis ===\n")
	sb.WriteString(rootCauseHint + "\n")

	sb.WriteString("\n=== Recommended Troubleshooting ===\n")
	sb.WriteString(checklist + "\n")

	if hr != nil {
		snapshot := BuildDiagnosticSnapshot(hr)
		if snapshot != "" {
			sb.WriteString("\n" + snapshot)
		}
	}

	return strings.TrimSpace(sb.String())
}

func formatTargetDownDescription(m *entities.Monitor, failCount int64, res *http.Result, lastErr error) string {
	var sb strings.Builder

	targetURL := ""
	method := "GET"
	if len(m.Steps) > 0 {
		targetURL = m.Steps[0].URL
		method = m.Steps[0].Method
	}

	sb.WriteString(fmt.Sprintf("Target service \"%s\" is DOWN and unreachable.\n\n", m.Name))
	sb.WriteString("=== Failure Summary ===\n")
	sb.WriteString(fmt.Sprintf("• Target Endpoint:          %s %s\n", method, targetURL))
	sb.WriteString(fmt.Sprintf("• Consecutive Failures:     %d (Threshold: 3)\n", failCount))
	if lastErr != nil {
		sb.WriteString(fmt.Sprintf("• Underlying Network Error: %s\n", lastErr.Error()))
	}
	if res != nil && res.Response.StatusCode > 0 {
		sb.WriteString(fmt.Sprintf("• Last HTTP Status:         %d (%s)\n", res.Response.StatusCode, xhttp.StatusText(res.Response.StatusCode)))
	} else {
		sb.WriteString("• Connection Status:        TCP / TLS Connection Failed (No HTTP response)\n")
	}

	sb.WriteString("\n=== Root-Cause Diagnosis ===\n")
	if lastErr != nil {
		errStr := strings.ToLower(lastErr.Error())
		if strings.Contains(errStr, "connection refused") {
			sb.WriteString("Port Closed / Process Down: Host responded with TCP RST. The target web service or container is not actively listening on the target port.\n")
		} else if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "lookup") {
			sb.WriteString("DNS Resolution Failure: Target hostname cannot be resolved. Check DNS records (A/AAAA/CNAME) and domain registrar status.\n")
		} else if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
			sb.WriteString("Network Timeout / Dropped Packets: Probe request timed out. The host may be offline, overloaded, or firewalled (SYN packets dropped).\n")
		} else if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "handshake") {
			sb.WriteString("TLS Handshake Failure: Secure connection could not be negotiated due to certificate verification or cipher mismatch.\n")
		} else {
			sb.WriteString(fmt.Sprintf("Network I/O Failure: %s\n", lastErr.Error()))
		}
	} else {
		sb.WriteString("The host did not respond to multiple consecutive probe attempts from distributed vantage points.\n")
	}

	sb.WriteString("\n=== Recommended Troubleshooting ===\n")
	sb.WriteString("1. Check service status: Verify application process / Docker container / Kubernetes pod is Running.\n")
	sb.WriteString("2. Check host reachability: Run `ping <host>` and `nc -zv <host> <port>` from external network.\n")
	sb.WriteString("3. Check ingress & firewalls: Inspect security groups, load balancers, and WAF rules.\n")
	sb.WriteString("4. Review system health: Inspect server CPU, memory, OOM killer logs, and disk space.\n")

	if res != nil && res.Response.StatusCode > 0 {
		snapshot := BuildDiagnosticSnapshot(res)
		if snapshot != "" {
			sb.WriteString("\n" + snapshot)
		}
	}

	return strings.TrimSpace(sb.String())
}

func formatAuthFailedDescription(m *entities.Monitor, res *http.Result, authErr error) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Authentication failed for monitor \"%s\".\n\n", m.Name))
	sb.WriteString("=== Authentication Details ===\n")
	sb.WriteString(fmt.Sprintf("• Auth Method:   %s\n", m.Settings.Auth.Method))
	if m.Settings.Auth.Method == "OAUTH2_CLIENT_CREDENTIALS" {
		sb.WriteString(fmt.Sprintf("• Token URL:     %s\n", m.Settings.Auth.TokenURL))
		sb.WriteString(fmt.Sprintf("• Client ID:     %s\n", m.Settings.Auth.ClientID))
	}
	if authErr != nil {
		sb.WriteString(fmt.Sprintf("• Error Detail:  %s\n", authErr.Error()))
	}

	sb.WriteString("\n=== Root-Cause Diagnosis ===\n")
	if authErr != nil {
		errStr := strings.ToLower(authErr.Error())
		if strings.Contains(errStr, "invalid_client") || strings.Contains(errStr, "401") {
			sb.WriteString("Invalid Client Credentials: The client_id or client_secret was rejected by the identity provider.\n")
		} else if strings.Contains(errStr, "invalid_scope") {
			sb.WriteString("Invalid Scope: The requested OAuth2 scope is not authorized for this client credentials grant.\n")
		} else if strings.Contains(errStr, "unsupported_grant_type") {
			sb.WriteString("Unsupported Grant Type: The identity provider endpoint does not support client_credentials grant.\n")
		} else {
			sb.WriteString(fmt.Sprintf("Identity Provider Communication Error: %s\n", authErr.Error()))
		}
	} else {
		sb.WriteString("Authentication credentials were rejected during probe execution.\n")
	}

	sb.WriteString("\n=== Recommended Troubleshooting ===\n")
	sb.WriteString("1. Verify that the Client ID and Client Secret configured in Opsway are valid and active.\n")
	if m.Settings.Auth.Method == "OAUTH2_CLIENT_CREDENTIALS" {
		sb.WriteString(fmt.Sprintf("2. Test token retrieval manually:\n   curl -X POST \"%s\" -d \"grant_type=client_credentials&client_id=%s&client_secret=*****\"\n", m.Settings.Auth.TokenURL, m.Settings.Auth.ClientID))
	} else {
		sb.WriteString("2. Verify credentials with origin system authentication provider.\n")
	}
	sb.WriteString("3. Check identity provider audit logs for rejected authentication events.\n")

	if res != nil {
		snapshot := BuildDiagnosticSnapshot(res)
		if snapshot != "" {
			sb.WriteString("\n" + snapshot)
		}
	}

	return strings.TrimSpace(sb.String())
}

func formatAnomalyDescription(m *entities.Monitor, res *http.Result, anomalyResp *ForecasterPredictResponse) string {
	var sb strings.Builder

	expected := 0.0
	upperBound := 0.0
	if anomalyResp != nil && len(anomalyResp.Predictions) > 0 {
		expected = anomalyResp.Predictions[0]
	}
	if anomalyResp != nil && len(anomalyResp.UpperBounds) > 0 {
		upperBound = anomalyResp.UpperBounds[0]
	}
	actual := float64(res.Timing.Phases.Total.Milliseconds())
	deviationPct := 0.0
	if expected > 0 {
		deviationPct = ((actual - expected) / expected) * 100.0
	}

	sb.WriteString(fmt.Sprintf("Latency anomaly detected for monitor \"%s\"!\n\n", m.Name))
	sb.WriteString("=== Latency Comparison ===\n")
	sb.WriteString(fmt.Sprintf("• Expected Baseline Latency: %.0f ms\n", expected))
	sb.WriteString(fmt.Sprintf("• Anomaly Threshold (Upper): %.0f ms\n", upperBound))
	sb.WriteString(fmt.Sprintf("• Observed Response Time:    %.0f ms (+%.1f%% above baseline)\n", actual, deviationPct))

	sb.WriteString("\n=== Timing Breakdown ===\n")
	sb.WriteString(fmt.Sprintf("• DNS Lookup:        %v\n", res.Timing.Phases.DNSLookup))
	sb.WriteString(fmt.Sprintf("• TCP Connection:    %v\n", res.Timing.Phases.TCPConnection))
	sb.WriteString(fmt.Sprintf("• TLS Handshake:     %v\n", res.Timing.Phases.TLSHandshake))
	sb.WriteString(fmt.Sprintf("• Server Processing: %v\n", res.Timing.Phases.ServerProcessing))
	sb.WriteString(fmt.Sprintf("• Content Transfer:  %v\n", res.Timing.Phases.ContentTransfer))

	sb.WriteString("\n=== Latency Bottleneck Diagnosis ===\n")
	if actual > 0 {
		spPct := float64(res.Timing.Phases.ServerProcessing.Milliseconds()) / actual * 100.0
		if spPct >= 50.0 {
			sb.WriteString(fmt.Sprintf("Server Processing (TTFB) accounted for %.1f%% of overall latency. The bottleneck is on the backend application or database.\n", spPct))
		} else if res.Timing.Phases.DNSLookup > 200*time.Millisecond {
			sb.WriteString(fmt.Sprintf("DNS Lookup latency was unusually elevated (%v). Check upstream DNS resolver performance.\n", res.Timing.Phases.DNSLookup))
		} else if res.Timing.Phases.TCPConnection > 200*time.Millisecond {
			sb.WriteString(fmt.Sprintf("TCP Connection latency was elevated (%v). Network congestion or packet retransmission detected.\n", res.Timing.Phases.TCPConnection))
		} else {
			sb.WriteString("Latency was distributed across multiple connection and data transfer phases.\n")
		}
	}

	sb.WriteString("\n=== Recommended Troubleshooting ===\n")
	sb.WriteString("1. Inspect backend application APM traces and slow database queries during the spike window.\n")
	sb.WriteString("2. Check server CPU utilization, memory pressure, and garbage collection frequency.\n")
	sb.WriteString("3. Review upstream dependencies or third-party API calls made during request processing.\n")

	snapshot := BuildDiagnosticSnapshot(res)
	if snapshot != "" {
		sb.WriteString("\n" + snapshot)
	}

	return strings.TrimSpace(sb.String())
}

func formatDeprecatedTLSDescription(res *http.Result) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Monitored server negotiated insecure %s (RFC 8996 Deprecated).\n\n", res.TLS.Version))
	sb.WriteString("=== TLS Negotiation Details ===\n")
	sb.WriteString(fmt.Sprintf("• Negotiated Version:  %s\n", res.TLS.Version))
	sb.WriteString(fmt.Sprintf("• Negotiated Cipher:   %s\n", res.TLS.Cipher))
	sb.WriteString(fmt.Sprintf("• Certificate Subject: %s\n", res.TLS.Certificate.Subject.CommonName))
	sb.WriteString(fmt.Sprintf("• Certificate Issuer:  %s\n", res.TLS.Certificate.Issuer.Organization))

	sb.WriteString("\n=== Security Risk & Compliance ===\n")
	sb.WriteString("TLS 1.0 and TLS 1.1 contain severe cryptographic design flaws (e.g., POODLE, BEAST) and have been formally deprecated under IETF RFC 8996, PCI-DSS v4.0, and NIST SP 800-52r2. Monitored connections must enforce at least TLS 1.2 or TLS 1.3.\n")

	sb.WriteString("\n=== Remediation Steps ===\n")
	sb.WriteString("1. Update web server or reverse proxy TLS configuration (e.g. Nginx `ssl_protocols TLSv1.2 TLSv1.3;`).\n")
	sb.WriteString("2. Disable legacy cipher suites and reload web server daemon.\n")
	sb.WriteString("3. Run `sslscan` or `testssl.sh` against the host to verify TLS 1.0/1.1 are no longer accepted.\n")

	return strings.TrimSpace(sb.String())
}

func formatCertChainInvalidDescription(res *http.Result, isHostnameMismatch bool, targetURL string) string {
	var sb strings.Builder
	if isHostnameMismatch {
		sb.WriteString("SSL/TLS Certificate Hostname Mismatch: Monitored hostname does not match certificate subject or SANs.\n\n")
	} else {
		sb.WriteString("SSL/TLS Certificate Chain Invalid: Certificate was not issued by a trusted Certificate Authority or chain is incomplete.\n\n")
	}

	sb.WriteString("=== Certificate Details ===\n")
	sb.WriteString(fmt.Sprintf("• Target Endpoint:     %s\n", targetURL))
	sb.WriteString(fmt.Sprintf("• Common Name (CN):    %s\n", res.TLS.Certificate.Subject.CommonName))
	sb.WriteString(fmt.Sprintf("• Issuer Organization: %s\n", res.TLS.Certificate.Issuer.Organization))
	sb.WriteString(fmt.Sprintf("• Valid Period:        %s to %s\n", res.TLS.Certificate.NotBefore.Format("2006-01-02"), res.TLS.Certificate.NotAfter.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("• Host Valid Match:    %v\n", res.TLS.Certificate.HostValid))
	sb.WriteString(fmt.Sprintf("• Trusted CA Chain:    %v\n", res.TLS.Certificate.TrustedCA))

	sb.WriteString("\n=== Recommended Troubleshooting ===\n")
	if isHostnameMismatch {
		sb.WriteString("1. Verify the SSL certificate's Subject Alternative Names (SANs) include the target domain.\n")
		sb.WriteString("2. Check for SNI (Server Name Indication) routing misconfiguration on reverse proxy or CDN.\n")
	} else {
		sb.WriteString("1. Verify that intermediate CA certificates are bundled in the web server's fullchain.pem.\n")
		sb.WriteString("2. If using a self-signed or private CA certificate, ensure it is installed in system trust store.\n")
		sb.WriteString("3. Ensure certificate was not revoked by the issuing CA via OCSP / CRL.\n")
	}

	return strings.TrimSpace(sb.String())
}

func formatCertExpiryDescription(res *http.Result, isExpired bool, daysRemaining int) string {
	var sb strings.Builder
	if isExpired {
		sb.WriteString(fmt.Sprintf("CRITICAL: SSL/TLS certificate for \"%s\" has EXPIRED!\n\n", res.TLS.Certificate.Subject.CommonName))
	} else {
		sb.WriteString(fmt.Sprintf("WARNING: SSL/TLS certificate for \"%s\" will expire in %d days.\n\n", res.TLS.Certificate.Subject.CommonName, daysRemaining))
	}

	sb.WriteString("=== Certificate Validity ===\n")
	sb.WriteString(fmt.Sprintf("• Certificate Subject: %s\n", res.TLS.Certificate.Subject.CommonName))
	sb.WriteString(fmt.Sprintf("• Issuer:              %s\n", res.TLS.Certificate.Issuer.Organization))
	sb.WriteString(fmt.Sprintf("• Valid From:          %s\n", res.TLS.Certificate.NotBefore.Format("2006-01-02 15:04:05 MST")))
	sb.WriteString(fmt.Sprintf("• Expiration Date:     %s\n", res.TLS.Certificate.NotAfter.Format("2006-01-02 15:04:05 MST")))
	if isExpired {
		sb.WriteString(fmt.Sprintf("• Expired Since:       %v ago\n", time.Since(res.TLS.Certificate.NotAfter).Round(time.Hour)))
	} else {
		sb.WriteString(fmt.Sprintf("• Days Remaining:      %d days\n", daysRemaining))
	}

	sb.WriteString("\n=== Remediation Steps ===\n")
	sb.WriteString("1. Trigger certificate renewal via automated ACME client (Certbot, cert-manager, or cloud provider ACM).\n")
	sb.WriteString("2. Deploy renewed full-chain certificate to ingress controllers or web servers.\n")
	sb.WriteString("3. Gracefully reload server process (e.g. `nginx -s reload`) to apply updated certificate.\n")

	return strings.TrimSpace(sb.String())
}
