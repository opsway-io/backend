package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	xhttp "net/http"
	"strings"
	"time"

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

	var res *http.Result
	var err error

	timeout := time.Duration(time.Second * 5)

	switch m.Settings.Method {
	case "TCP":
		res, err = tcpProber.Probe(ctx, m.Settings.URL, timeout)
	case "WEBSOCKET":
		res, err = websocketProber.Probe(ctx, m.Settings.URL, timeout)
	case "UDP":
		res, err = udpProber.Probe(ctx, m.Settings.URL, timeout)
	case "ICMP":
		res, err = icmpProber.Probe(ctx, m.Settings.URL, timeout)
	case "DNS":
		res, err = dnsProber.Probe(ctx, m.Settings.URL, timeout)
	case "POSTGRES":
		res, err = postgresProber.Probe(ctx, m.Settings.URL, timeout)
	case "MYSQL":
		res, err = mysqlProber.Probe(ctx, m.Settings.URL, timeout)
	case "REDIS":
		res, err = redisProber.Probe(ctx, m.Settings.URL, timeout)
	case "BROWSER":
		var scriptJSON string
		if m.Settings.Body.Content != nil {
			scriptJSON = string(*m.Settings.Body.Content)
		}
		// Browser needs longer timeout, give it 15 seconds
		browserTimeout := time.Duration(time.Second * 15)
		res, err = browserProber.Probe(ctx, m.Settings.URL, scriptJSON, browserTimeout)
	default:
		res, err = httpProber.Probe(
			ctx,
			m.Settings.Method,
			m.Settings.URL,
			nil,
			nil,
			timeout,
		)
	}

	if err != nil && res == nil {
		l.WithError(err).Error("failed to probe")

		return
	}

	l = l.WithFields(logrus.Fields{
		"status":     res.Response.StatusCode,
		"total_time": fmt.Sprintf("%v", res.Timing.Phases.Total),
	})

	newCheck := mapResultToCheck(m, res, location)

	if err = c.Create(ctx, newCheck); err != nil {
		l.WithError(err).Error("failed add result to clickhouse")

		return
	}

	failed, passed, err := assertResult(res, m.Assertions)
	if err != nil {
		l.WithError(err).Error("failed to assert result")

		return
	}

	failedCount := len(failed)
	passedCount := len(passed)

	l = l.WithFields(logrus.Fields{
		"assertions_passed": passedCount,
		"assertions_failed": failedCount,
	})

	failKey := fmt.Sprintf("monitor:%d:failures", m.ID)

	if failedCount > 0 {
		l.Info("some assertions failed, incrementing failure counter")
		val, err := rc.Incr(ctx, failKey).Result()
		if err != nil {
			l.WithError(err).Error("failed to increment failure counter")
		}

		// Hardcoded threshold of 3 for MVP
		if val == 3 {
			l.Info("failure threshold reached, triggering incident")
			if err = triggerIncident(ctx, m, res, &failed, i); err != nil {
				l.WithError(err).Error("failed to trigger incident")
			}
		} else if val > 3 {
			// Already triggered, could optionally update the incident here
		}
	} else {
		l.Info("all assertions passed")

		// Reset counter
		rc.Del(ctx, failKey)

		// Auto-resolve any open incidents for this monitor
		openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
		if err == nil && openIncidents != nil {
			for _, inc := range *openIncidents {
				if !inc.Incident.Resolved && inc.Incident.Title != "Anomaly Detected" && inc.Incident.Title != "SSL/TLS Cert Expiry" {
					l.WithField("incident_id", inc.Incident.ID).Info("auto-resolving incident")
					inc.Incident.Resolved = true
					if err := i.Update(ctx, &inc.Incident); err != nil {
						l.WithError(err).Error("failed to resolve incident")
					}
				}
			}
		}
	}

	// Check for anomalies
	isAnomaly, err := checkAnomaly(m.ID, res)
	if err != nil {
		l.WithError(err).Error("failed to check for anomaly")
	} else if isAnomaly {
		hasOpenAnomalyIncident := false
		openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
		if err == nil && openIncidents != nil {
			for _, inc := range *openIncidents {
				if inc.Title == "Anomaly Detected" {
					hasOpenAnomalyIncident = true
					break
				}
			}
		}

		if !hasOpenAnomalyIncident {
			l.Info("anomaly detected, triggering incident")
			desc := "Response time anomaly detected by forecaster"
			anomalyIncident := entities.Incident{
				MonitorID:   &m.ID,
				TeamID:      m.TeamID,
				Title:       "Anomaly Detected",
				Description: &desc,
			}
			if err := i.Create(ctx, &[]entities.Incident{anomalyIncident}); err != nil {
				l.WithError(err).Error("failed to trigger anomaly incident")
			}
		}
	}

	// SSL/TLS Certificate Expiration Monitoring
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
				desc := fmt.Sprintf("SSL/TLS certificate for %s expires in %.1f days (on %s)",
					m.Settings.URL,
					timeRemaining.Hours()/24,
					expiry.Format(time.RFC822),
				)
				sslIncident := entities.Incident{
					MonitorID:   &m.ID,
					TeamID:      m.TeamID,
					Title:       "SSL/TLS Cert Expiry",
					Description: &desc,
				}
				if err := i.Create(ctx, &[]entities.Incident{sslIncident}); err != nil {
					l.WithError(err).Error("failed to trigger SSL/TLS cert expiry incident")
				}
			}
		} else {
			// Auto-resolve any open SSL/TLS cert expiry incidents if the cert is now valid for >= 30 days
			openIncidents, err := i.GetByMonitorIDWithAssertionPaginated(ctx, m.ID, nil, nil)
			if err == nil && openIncidents != nil {
				for _, inc := range *openIncidents {
					if inc.Title == "SSL/TLS Cert Expiry" {
						l.Info("SSL/TLS certificate is now valid, resolving open incident")
						inc.Incident.Resolved = true
						if err := i.Update(ctx, &inc.Incident); err != nil {
							l.WithError(err).Error("failed to resolve SSL/TLS cert expiry incident")
						}
					}
				}
			}
		}
	}
}

func checkAnomaly(monitorID uint, res *http.Result) (bool, error) {
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
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != xhttp.StatusOK {
		return false, fmt.Errorf("forecaster API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(body), `"anomalies":[true]`) || strings.Contains(string(body), `"anomalies": [true]`), nil
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
		Method:     m.Settings.Method,
		URL:        m.Settings.URL,
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

		incidents[j] = entities.Incident{
			MonitorID:          &assertion.MonitorID,
			TeamID:             m.TeamID,
			Title:              assertion.Source,
			Description:        &assertion.Source,
			MonitorAssertionID: &assertion.ID,
		}
	}

	return i.Upsert(ctx, &incidents)
}
