package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	xhttp "net/http"
	"strings"
	"time"

	"github.com/yalp/jsonpath"

	"github.com/gammazero/workerpool"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	connectorRedis "github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/probes/browser"
	"github.com/opsway-io/backend/internal/probes/dns"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/opsway-io/backend/internal/probes/http/asserter"
	"github.com/opsway-io/backend/internal/probes/icmp"
	probeMysql "github.com/opsway-io/backend/internal/probes/mysql"
	probePostgres "github.com/opsway-io/backend/internal/probes/postgres"
	probeRedis "github.com/opsway-io/backend/internal/probes/redis"
	"github.com/opsway-io/backend/internal/probes/tcp"
	"github.com/opsway-io/backend/internal/probes/udp"
	"github.com/opsway-io/backend/internal/probes/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/spf13/cobra"
)

type ProberConfig struct {
	Concurrency        int      `mapstructure:"concurrency" default:"25"`
	Location           string   `mapstructure:"location" default:"global"`
	AvailableLocations []string `mapstructure:"available_locations"`
}

//nolint:gochecknoglobals
var proberCmd = &cobra.Command{
	Use: "prober",
	Run: runProber,
}

var asserterInst = asserter.New()

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(proberCmd)
}

func runProber(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	conf, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	wp := workerpool.New(conf.Prober.Concurrency)

	l := getLogger(conf.Log)

	l.WithFields(logrus.Fields{
		"host": conf.Redis.Host,
		"port": conf.Redis.Port,
		"db":   conf.Redis.DB,
	}).Info("Connecting to redis")

	redisClient, err := connectorRedis.NewClient(ctx, conf.Redis)
	if err != nil {
		l.WithError(err).Fatal("failed to connect to redis")
	}

	ch, err := clickhouse.NewClient(ctx, conf.Clickhouse)
	if err != nil {
		l.WithError(err).Fatal("Failed to create clickhouse")
	}

	httpResultService := check.NewService(ch)

	db, err := postgres.NewClient(ctx, conf.Postgres)
	if err != nil {
		l.WithError(err).Fatal("Failed to create Postgres client")
	}

	eventService, err := event.NewService(redisClient, "prober")
	if err != nil {
		l.WithError(err).Fatal("Failed to create event service")
	}

	incidentRepository := incident.NewRepository(db)
	incidentService := incident.NewService(incidentRepository, eventService)

	httpProber := http.NewService(conf.HTTPProbe)
	tcpProber := tcp.NewService()
	icmpProber := icmp.NewService()
	dnsProber := dns.NewService()
	postgresProber := probePostgres.NewService()
	mysqlProber := probeMysql.NewService()
	redisProber := probeRedis.NewService()
	browserProber := browser.NewService()
	websocketProber := websocket.NewService()
	udpProber := udp.NewService()

	l.Info("Waiting for tasks...")

	subscriber, err := eventService.Subscribe(ctx, fmt.Sprintf("prober.tasks.%s", conf.Prober.Location))
	if err != nil {
		l.WithError(err).Fatal("failed to subscribe to prober tasks")
	}

	for {
		select {
		case <-ctx.Done():
			l.Info("Shutting down...")
			wp.StopWait()
			l.Info("Goodbye!")
			return
		case msg := <-subscriber:
			if msg == nil {
				continue
			}

			wp.Submit(func() {
				var task events.ProberTask
				if err := json.Unmarshal(msg.Payload, &task); err != nil {
					l.WithError(err).Error("failed to unmarshal prober task")
					msg.Nack()
					return
				}

				handleTask(ctx, l, httpProber, tcpProber, icmpProber, dnsProber, postgresProber, mysqlProber, redisProber, browserProber, websocketProber, udpProber, task.Monitor, httpResultService, incidentService, conf.Prober.Location, redisClient)
				msg.Ack()
			})
		}
	}

	wp.StopWait()

	l.Info("Goodbye!")
}

func handleTask(ctx context.Context, logger *logrus.Logger, httpProber http.Service, tcpProber tcp.Service, icmpProber icmp.Service, dnsProber dns.Service, postgresProber probePostgres.Service, mysqlProber probeMysql.Service, redisProber probeRedis.Service, browserProber browser.Service, websocketProber websocket.Service, udpProber udp.Service, m *entities.Monitor, c check.Service, i incident.Service, location string, rc *redis.Client) {
	l := logger.WithFields(logrus.Fields{
		"monitor_id": m.ID,
		"location":   location,
	})

	if len(m.Steps) == 0 {
		l.Error("monitor has no steps")
		return
	}

	timeout := time.Duration(time.Second * 5)
	extractedVariables := make(map[string]string)

	var overallRes *http.Result
	var err error
	var authFailed bool = false
	var targetDown bool = false
	var anyAssertionsFailed bool = false
	var allFailedAssertions []entities.MonitorAssertion
	
	// Create a combined timing accumulator
	totalDNSLookup := time.Duration(0)
	totalTCPConnection := time.Duration(0)
	totalTLSHandshake := time.Duration(0)
	totalServerProcessing := time.Duration(0)
	totalContentTransfer := time.Duration(0)
	totalTime := time.Duration(0)

	for stepIdx, step := range m.Steps {
		var res *http.Result

		maxRetries := 3
		for attempt := 0; attempt < maxRetries; attempt++ {
			err = nil
			authFailed = false

			stepURL := step.URL
			var stepBodyStr string
			if step.Body.Content != nil {
				stepBodyStr = string(*step.Body.Content)
			}
			
			for k, v := range extractedVariables {
				placeholder := fmt.Sprintf("{{%s}}", k)
				stepURL = strings.ReplaceAll(stepURL, placeholder, v)
				stepBodyStr = strings.ReplaceAll(stepBodyStr, placeholder, v)
			}
			
			var bodyReader io.Reader
			if stepBodyStr != "" {
				bodyReader = strings.NewReader(stepBodyStr)
			}

			switch step.Method {
			case "TCP":
				res, err = tcpProber.Probe(ctx, stepURL, timeout)
			case "WEBSOCKET":
				res, err = websocketProber.Probe(ctx, stepURL, timeout)
			case "UDP":
				res, err = udpProber.Probe(ctx, stepURL, timeout)
			case "ICMP":
				res, err = icmpProber.Probe(ctx, stepURL, timeout)
			case "DNS":
				res, err = dnsProber.Probe(ctx, stepURL, timeout)
			case "POSTGRES":
				res, err = postgresProber.Probe(ctx, stepURL, timeout)
			case "MYSQL":
				res, err = mysqlProber.Probe(ctx, stepURL, timeout)
			case "REDIS":
				res, err = redisProber.Probe(ctx, stepURL, timeout)
			case "BROWSER":
				browserTimeout := time.Duration(time.Second * 15)
				res, err = browserProber.Probe(ctx, stepURL, stepBodyStr, browserTimeout)
			default:
				headers := make(map[string]string)
				for _, h := range step.Headers {
					headers[h.Key] = h.Value
				}

				if m.Settings.Auth.Method == "BASIC" {
					auth := m.Settings.Auth.Username + ":" + m.Settings.Auth.Password
					headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
				} else if m.Settings.Auth.Method == "OAUTH2_CLIENT_CREDENTIALS" {
					token, authErr := fetchOAuth2Token(ctx, m.Settings.Auth.TokenURL, m.Settings.Auth.ClientID, m.Settings.Auth.ClientSecret)
					if authErr != nil {
						err = fmt.Errorf("failed to fetch oauth2 token: %w", authErr)
						authFailed = true
					} else {
						headers["Authorization"] = "Bearer " + token
					}
				}

				if err == nil {
					res, err = httpProber.Probe(
						ctx,
						step.Method,
						stepURL,
						headers,
						bodyReader,
						timeout,
					)
				}
			}

			if err == nil {
				break
			}

			if attempt < maxRetries-1 {
				time.Sleep(1 * time.Second)
			}
		}

		if err != nil && res == nil {
			l.WithError(err).Error("failed to probe step")
			if !authFailed {
				targetDown = true
			}

			res = &http.Result{
				Response: http.Response{
					StatusCode: 0,
					Header:     make(xhttp.Header),
					Body:       []byte{},
				},
				Timing: http.Timing{
					Phases: http.TimingPhases{
						Total:            timeout,
						ServerProcessing: timeout,
					},
				},
			}
		}

		overallRes = res

		// Accumulate timings
		totalDNSLookup += res.Timing.Phases.DNSLookup
		totalTCPConnection += res.Timing.Phases.TCPConnection
		totalTLSHandshake += res.Timing.Phases.TLSHandshake
		totalServerProcessing += res.Timing.Phases.ServerProcessing
		totalContentTransfer += res.Timing.Phases.ContentTransfer
		totalTime += res.Timing.Phases.Total

		if !targetDown && len(step.Variables) > 0 {
			var unmarshalData interface{}
			_ = json.Unmarshal(res.Response.Body, &unmarshalData)

			for _, v := range step.Variables {
				if v.Source == "JSON_BODY" && unmarshalData != nil {
					val, err := jsonpath.Read(unmarshalData, v.Property)
					if err == nil {
						extractedVariables[v.Name] = fmt.Sprintf("%v", val)
					}
				} else if v.Source == "HEADER" {
					val := res.Response.Header.Get(v.Property)
					if val != "" {
						extractedVariables[v.Name] = val
					}
				}
			}
		}

		var failed []entities.MonitorAssertion
		if !authFailed {
			failed, _, err = assertResult(res, step.Assertions)
			if err != nil {
				l.WithError(err).Error("failed to assert result")
			} else if len(failed) > 0 {
				anyAssertionsFailed = true
				allFailedAssertions = append(allFailedAssertions, failed...)
			}
		}

		// Abort subsequent steps if this step failed in any way
		if targetDown || authFailed || anyAssertionsFailed {
			l.WithField("failed_step_index", stepIdx).Info("Step failed, aborting sequence")
			break
		}
	}

	// Update overallRes timings to the sum
	if overallRes != nil {
		overallRes.Timing.Phases.DNSLookup = totalDNSLookup
		overallRes.Timing.Phases.TCPConnection = totalTCPConnection
		overallRes.Timing.Phases.TLSHandshake = totalTLSHandshake
		overallRes.Timing.Phases.ServerProcessing = totalServerProcessing
		overallRes.Timing.Phases.ContentTransfer = totalContentTransfer
		overallRes.Timing.Phases.Total = totalTime
	} else {
		// Should never happen if steps > 0
		return 
	}

	res := overallRes

	l = l.WithFields(logrus.Fields{
		"status":     res.Response.StatusCode,
		"total_time": fmt.Sprintf("%v", res.Timing.Phases.Total),
	})

	failedCount := len(allFailedAssertions)

	if !authFailed {
		newCheck := mapResultToCheck(m, res, location)
		if failedCount > 0 {
			newCheck.StatusCode = 0
		}

		if err = c.Create(ctx, newCheck); err != nil {
			l.WithError(err).Error("failed add result to clickhouse")
			return
		}
	}

	failKey := fmt.Sprintf("monitor:%d:failures", m.ID)
	var failCount int64 = 0

	if failedCount > 0 || targetDown || authFailed {
		l.Info("monitor failed, incrementing failure counter")
		val, err := rc.Incr(ctx, failKey).Result()
		if err != nil {
			l.WithError(err).Error("failed to increment failure counter")
		}
		failCount = val
	}

	if failedCount > 0 {
		if failCount == 3 {
			l.Info("failure threshold reached, triggering incident")
			if err = triggerIncident(ctx, m, res, &allFailedAssertions, i); err != nil {
				l.WithError(err).Error("failed to trigger incident")
			}
		} else if failCount > 3 {
			openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
			if err == nil && openIncidents != nil {
				var unhandledFailures []entities.MonitorAssertion
				openAssertionIDs := make(map[uint]bool)
				for _, inc := range *openIncidents {
					if inc.MonitorAssertionID != nil {
						openAssertionIDs[*inc.MonitorAssertionID] = true
					}
				}
				for _, f := range allFailedAssertions {
					if !openAssertionIDs[f.ID] {
						unhandledFailures = append(unhandledFailures, f)
					}
				}
				if len(unhandledFailures) > 0 {
					l.Info("some assertions are failing but have no open incidents (likely resolved manually), re-triggering")
					if err = triggerIncident(ctx, m, res, &unhandledFailures, i); err != nil {
						l.WithError(err).Error("failed to trigger incident")
					}
				}
			}
			openIncidents, err = i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
			if err == nil && openIncidents != nil {
				for _, inc := range *openIncidents {
					if !inc.Incident.Resolved && inc.Incident.Title != "Anomaly Detected" && inc.Incident.Title != "SSL/TLS Cert Expiry" && inc.Incident.Title != "Target Down" && inc.Incident.Title != "Authentication Failed" {
						for _, assertion := range allFailedAssertions {
							if inc.Incident.MonitorAssertionID != nil && *inc.Incident.MonitorAssertionID == assertion.ID {
								inc.Incident.UpdatedAt = time.Now()
								inc.Incident.Occurrences++
								if err := i.Update(ctx, &inc.Incident); err != nil {
									l.WithError(err).Error("failed to update incident occurrence time")
								}
								_ = i.CreateOccurrence(ctx, &entities.IncidentOccurrence{IncidentID: inc.Incident.ID, CreatedAt: time.Now()})
							}
						}
					}
				}
			}
		}
	} else {
		l.Info("all assertions passed")
		if !authFailed && !targetDown {
			rc.Del(ctx, failKey)
		}
	}

	openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
	if err == nil && openIncidents != nil {
		failedAssertionIDs := make(map[uint]bool)
		for _, f := range allFailedAssertions {
			failedAssertionIDs[f.ID] = true
		}

		for _, inc := range *openIncidents {
			if !inc.Incident.Resolved && inc.Incident.Title != "Anomaly Detected" && inc.Incident.Title != "SSL/TLS Cert Expiry" && inc.Incident.Title != "Target Down" && inc.Incident.Title != "Authentication Failed" {
				if inc.Incident.MonitorAssertionID == nil || !failedAssertionIDs[*inc.Incident.MonitorAssertionID] {
					l.WithField("incident_id", inc.Incident.ID).Info("auto-resolving incident")
					inc.Incident.Resolved = true
					now := time.Now()
					inc.Incident.ResolvedAt = &now
					if err := i.Update(ctx, &inc.Incident); err != nil {
						l.WithError(err).Error("failed to resolve incident")
					}
				}
			}
		}
	}

	if targetDown {
		hasOpenDownIncident := false
		openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
		if err == nil && openIncidents != nil {
			for _, inc := range *openIncidents {
				if inc.Title == "Target Down" {
					hasOpenDownIncident = true
					inc.Incident.UpdatedAt = time.Now()
					inc.Incident.Occurrences++
					if err := i.Update(ctx, &inc.Incident); err != nil {
						l.WithError(err).Error("failed to update target down incident occurrence")
					}
					_ = i.CreateOccurrence(ctx, &entities.IncidentOccurrence{IncidentID: inc.Incident.ID, CreatedAt: time.Now()})
					break
				}
			}
		}

		if !hasOpenDownIncident && failCount >= 3 {
			l.Info("target is down and failure threshold reached, triggering incident")
			desc := "Target is unreachable or not responding."
			downIncident := entities.Incident{
				MonitorID:   &m.ID,
				TeamID:      m.TeamID,
				Title:       "Target Down",
				Description: &desc,
			}
			if err := i.Create(ctx, &[]entities.Incident{downIncident}); err != nil {
				l.WithError(err).Error("failed to trigger target down incident")
			}
		}
	} else {
		openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
		if err == nil && openIncidents != nil {
			for _, inc := range *openIncidents {
				if inc.Title == "Target Down" && !inc.Resolved {
					l.Info("target is now responding, resolving open Target Down incident")
					inc.Incident.Resolved = true
					now := time.Now()
					inc.Incident.ResolvedAt = &now
					if err := i.Update(ctx, &inc.Incident); err != nil {
						l.WithError(err).Error("failed to resolve target down incident")
					}
				}
			}
		}

		anomalyResp, err := checkAnomaly(m.ID, res)
		if err != nil {
			l.WithError(err).Error("failed to check for anomaly")
		} else if anomalyResp != nil && len(anomalyResp.Anomalies) > 0 && anomalyResp.Anomalies[0] {
			hasOpenAnomalyIncident := false
			if openIncidents != nil {
				for _, inc := range *openIncidents {
					if inc.Title == "Anomaly Detected" && !inc.Resolved {
						hasOpenAnomalyIncident = true
						break
					}
				}
			}

			if !hasOpenAnomalyIncident {
				l.Info("anomaly detected, triggering incident")
				desc := fmt.Sprintf("Response time anomaly detected! Expected response time was %.0fms (upper limit: %.0fms), but actual response time was %.0fms.", anomalyResp.Predictions[0], anomalyResp.UpperBounds[0], float64(res.Timing.Phases.Total.Milliseconds()))
				anomalyIncident := entities.Incident{
					MonitorID:   &m.ID,
					TeamID:      m.TeamID,
					Title:       "Anomaly Detected",
					Description: &desc,
				}
				if err := i.Create(ctx, &[]entities.Incident{anomalyIncident}); err != nil {
					l.WithError(err).Error("failed to trigger anomaly incident")
				}
			} else {
				for _, inc := range *openIncidents {
					if inc.Title == "Anomaly Detected" && !inc.Resolved {
						inc.Incident.UpdatedAt = time.Now()
						inc.Incident.Occurrences++
						if err := i.Update(ctx, &inc.Incident); err != nil {
							l.WithError(err).Error("failed to update anomaly incident occurrence")
						}
						_ = i.CreateOccurrence(ctx, &entities.IncidentOccurrence{IncidentID: inc.Incident.ID, CreatedAt: time.Now()})
						break
					}
				}
			}
		}
	}

	if authFailed {
		hasOpenAuthIncident := false
		openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
		if err == nil && openIncidents != nil {
			for _, inc := range *openIncidents {
				if inc.Title == "Authentication Failed" && !inc.Resolved {
					hasOpenAuthIncident = true
					inc.Incident.UpdatedAt = time.Now()
					inc.Incident.Occurrences++
					if err := i.Update(ctx, &inc.Incident); err != nil {
						l.WithError(err).Error("failed to update auth failed incident occurrence")
					}
					_ = i.CreateOccurrence(ctx, &entities.IncidentOccurrence{IncidentID: inc.Incident.ID, CreatedAt: time.Now()})
					break
				}
			}
		}

		if !hasOpenAuthIncident && failCount >= 3 {
			l.Info("authentication failed and threshold reached, triggering incident")
			descStr := "Authentication failed"
			authIncident := entities.Incident{
				MonitorID:   &m.ID,
				TeamID:      m.TeamID,
				Title:       "Authentication Failed",
				Description: &descStr,
			}
			if err := i.Create(ctx, &[]entities.Incident{authIncident}); err != nil {
				l.WithError(err).Error("failed to trigger auth failed incident")
			}
		}
	} else {
		openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
		if err == nil && openIncidents != nil {
			for _, inc := range *openIncidents {
				if inc.Title == "Authentication Failed" && !inc.Resolved {
					l.Info("authentication is now succeeding, resolving open Authentication Failed incident")
					inc.Incident.Resolved = true
					now := time.Now()
					inc.Incident.ResolvedAt = &now
					if err := i.Update(ctx, &inc.Incident); err != nil {
						l.WithError(err).Error("failed to resolve auth failed incident")
					}
				}
			}
		}
	}

	if res.TLS != nil {
		expiry := res.TLS.Certificate.NotAfter
		timeRemaining := time.Until(expiry)

		if timeRemaining > 0 && timeRemaining < 30*24*time.Hour {
			hasOpenSSLIncident := false
			openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
			if err == nil && openIncidents != nil {
				for _, inc := range *openIncidents {
					if inc.Title == "SSL/TLS Cert Expiry" {
						hasOpenSSLIncident = true
						break
					}
				}
			}

			if !hasOpenSSLIncident {
				l.Warn("SSL/TLS certificate is expiring soon, triggering incident")
				// just simple desc
				desc := "SSL/TLS certificate is expiring soon"
				sslIncident := entities.Incident{
					MonitorID:   &m.ID,
					TeamID:      m.TeamID,
					Title:       "SSL/TLS Cert Expiry",
					Description: &desc,
				}
				if err := i.Create(ctx, &[]entities.Incident{sslIncident}); err != nil {
					l.WithError(err).Error("failed to trigger SSL/TLS cert expiry incident")
				}
			} else {
				for _, inc := range *openIncidents {
					if inc.Title == "SSL/TLS Cert Expiry" {
						inc.Incident.UpdatedAt = time.Now()
						if err := i.Update(ctx, &inc.Incident); err != nil {
							l.WithError(err).Error("failed to update ssl incident occurrence")
						}
						break
					}
				}
			}
		} else {
			openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
			if err == nil && openIncidents != nil {
				for _, inc := range *openIncidents {
					if inc.Title == "SSL/TLS Cert Expiry" {
						l.Info("SSL/TLS certificate is now valid, resolving open incident")
						inc.Incident.Resolved = true
						now := time.Now()
						inc.Incident.ResolvedAt = &now
						if err := i.Update(ctx, &inc.Incident); err != nil {
							l.WithError(err).Error("failed to resolve SSL/TLS cert expiry incident")
						}
					}
				}
			}
		}
	}
}


type ForecasterPredictResponse struct {
	Anomalies   []bool    `json:"anomalies"`
	Predictions []float64 `json:"predictions"`
	UpperBounds []float64 `json:"upper_bounds"`
	LowerBounds []float64 `json:"lower_bounds"`
}

func checkAnomaly(monitorID uint, res *http.Result) (*ForecasterPredictResponse, error) {
	reqBody := fmt.Sprintf(`{"monitor_id": %d, "timings": [{"response_time": %f, "dns_lookup": %f, "tcp_connection": %f, "tls_handshake": %f, "server_processing": %f, "content_transfer": %f}]}`,
		monitorID,
		float64(res.Timing.Phases.Total.Milliseconds()),
		float64(res.Timing.Phases.DNSLookup.Milliseconds()),
		float64(res.Timing.Phases.TCPConnection.Milliseconds()),
		float64(res.Timing.Phases.TLSHandshake.Milliseconds()),
		float64(res.Timing.Phases.ServerProcessing.Milliseconds()),
		float64(res.Timing.Phases.ContentTransfer.Milliseconds()),
	)
	resp, err := xhttp.Post("http://forecaster-api:8000/predict", "application/json", strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != xhttp.StatusOK {
		return nil, fmt.Errorf("forecaster API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed ForecasterPredictResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	return &parsed, nil
}

func fetchOAuth2Token(ctx context.Context, tokenURL, clientID, clientSecret string) (string, error) {
	if !strings.HasPrefix(tokenURL, "http://") && !strings.HasPrefix(tokenURL, "https://") {
		tokenURL = "https://" + tokenURL
	}

	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
	}

	client := &xhttp.Client{Timeout: 5 * time.Second}
	ctxWithClient := context.WithValue(ctx, oauth2.HTTPClient, client)

	token, err := conf.Token(ctxWithClient)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

func assertResult(httpResult *http.Result, assertions []entities.MonitorAssertion) ([]entities.MonitorAssertion, []entities.MonitorAssertion, error) {
	if len(assertions) == 0 {
		return nil, nil, nil
	}

	rules := mapMonitorAssertionsToAssertionRules(assertions)

	assertResult, err := asserterInst.Assert(httpResult, rules)
	if err != nil {
		return nil, nil, err
	}

	failed := []entities.MonitorAssertion{}
	passed := []entities.MonitorAssertion{}

	for i, ok := range assertResult {
		if ok {
			passed = append(passed, assertions[i])
		} else {
			failed = append(failed, assertions[i])
		}
	}

	return failed, passed, nil
}

func mapMonitorAssertionsToAssertionRules(ma []entities.MonitorAssertion) []asserter.Rule {
	rules := make([]asserter.Rule, len(ma))

	for i, assertion := range ma {
		rules[i] = asserter.Rule{
			Source:   assertion.Source,
			Operator: assertion.Operator,
			Property: assertion.Property,
			Target:   assertion.Target,
		}
	}

	return rules
}

func mapResultToCheck(m *entities.Monitor, res *http.Result, location string) *check.Check {
	c := &check.Check{
		MonitorID:  uint64(m.ID),
		TeamID:     uint64(m.TeamID),
		StatusCode: uint64(res.Response.StatusCode),
		Method:     m.Steps[0].Method,
		URL:        m.Steps[0].URL,
		Location:   location,
		Timing: check.Timing{
			DNSLookup:        res.Timing.Phases.DNSLookup,
			TCPConnection:    res.Timing.Phases.TCPConnection,
			TLSHandshake:     res.Timing.Phases.TLSHandshake,
			ServerProcessing: res.Timing.Phases.ServerProcessing,
			ContentTransfer:  res.Timing.Phases.ContentTransfer,
			Total:            res.Timing.Phases.Total,
		},
	}

	if res.TLS != nil {
		c.TLS = &check.TLS{
			Version:   res.TLS.Version,
			Cipher:    res.TLS.Cipher,
			Issuer:    res.TLS.Certificate.Issuer.Organization,
			Subject:   res.TLS.Certificate.Subject.CommonName,
			NotBefore: res.TLS.Certificate.NotBefore,
			NotAfter:  res.TLS.Certificate.NotAfter,
		}
	}

	return c
}

func triggerIncident(ctx context.Context, m *entities.Monitor, hr *http.Result, failed *[]entities.MonitorAssertion, i incident.Service) error {
	incidents := make([]entities.Incident, len(*failed))

	for j := range *failed {
		assertion := (*failed)[j]

		desc := fmt.Sprintf("Assertion failed: %s %s %s", assertion.Source, assertion.Operator, assertion.Target)
		if assertion.Property != "" {
			desc = fmt.Sprintf("Assertion failed: %s (%s) %s %s", assertion.Source, assertion.Property, assertion.Operator, assertion.Target)
		}

		incidents[j] = entities.Incident{
			MonitorID:          &assertion.MonitorID,
			TeamID:             m.TeamID,
			Title:              assertion.Source,
			Description:        &desc,
			MonitorAssertionID: &assertion.ID,
			Occurrences:        1,
		}
	}

	err := i.Upsert(ctx, &incidents)
	if err == nil {
		for _, inc := range incidents {
			_ = i.CreateOccurrence(ctx, &entities.IncidentOccurrence{IncidentID: inc.ID, CreatedAt: time.Now()})
		}
	}
	return err
}
