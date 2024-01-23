package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	connectorRedis "github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/opsway-io/backend/internal/probes/http/asserter"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

type ProberConfig struct {
	Concurrency int `mapstructure:"concurrency" default:"25"`
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

	schedule := monitor.NewSchedule(redisClient)

	ch, err := clickhouse.NewClient(ctx, conf.Clickhouse)
	if err != nil {
		l.WithError(err).Fatal("Failed to create clickhouse")
	}

	httpResultService := check.NewService(ch)

	prober := http.NewService(conf.HTTPProbe)

	l.Info("Waiting for tasks...")

	if err := schedule.On(ctx, func(ctx context.Context, monitor *entities.Monitor) {
		wp.Submit(func() {
			handleTask(ctx, l, prober, monitor, httpResultService)
		})
	}); err != nil {
		l.WithError(err).Fatal("failed to start schedule")
	}

	l.Info("Shutting down...")

	wp.StopWait()

	l.Info("Goodbye!")
}

func handleTask(ctx context.Context, logger *logrus.Logger, prober http.Service, m *entities.Monitor, c check.Service) {
	l := logger.WithFields(logrus.Fields{
		"monitor_id": m.ID,
	})

	res, err := prober.Probe(
		ctx,
		m.Settings.Method,
		m.Settings.URL,
		nil,
		nil,
		time.Duration(time.Second*5),
	)
	if err != nil {
		l.WithError(err).Error("failed to probe")

		return
	}

	l = l.WithFields(logrus.Fields{
		"status":     res.Response.StatusCode,
		"total_time": fmt.Sprintf("%v", res.Timing.Phases.Total),
	})

	newCheck := mapResultToCheck(m, res)

	err = c.Create(ctx, newCheck)
	if err != nil {
		l.WithError(err).Error("failed add result to clickhouse")
	}

	failed, passed, err := assertResult(res, m.Assertions)
	if err != nil {
		l.WithError(err).Error("failed to assert result")

		return
	}

	failedCount := len(*failed)
	passedCount := len(*passed)

	l = l.WithFields(logrus.Fields{
		"assertions_passed": passedCount,
		"assertions_failed": failedCount,
	})

	if failedCount > 0 {
		l.Info("some assertions failed, triggering incident")

		if err = triggerIncident(m, res, failed); err != nil {
			l.WithError(err).Error("failed to trigger incident")
		}
	} else {
		l.Info("all assertions passed")
	}
}

func assertResult(httpResult *http.Result, assertions []entities.MonitorAssertion) (failed, passed *[]entities.MonitorAssertion, err error) {
	if len(assertions) == 0 {
		return nil, nil, nil
	}

	rules := mapMonitorAssertionsToAssertionRules(assertions)

	assertResult, err := asserterInst.Assert(httpResult, rules)
	if err != nil {
		return nil, nil, err
	}

	failed = &[]entities.MonitorAssertion{}
	passed = &[]entities.MonitorAssertion{}

	for i, ok := range assertResult {
		if ok {
			*passed = append(*passed, assertions[i])
		} else {
			*failed = append(*failed, assertions[i])
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
			Property: assertion.Target,
			Target:   assertion.Property,
		}
	}

	return rules
}

func mapResultToCheck(m *entities.Monitor, res *http.Result) *check.Check {
	c := &check.Check{
		MonitorID:  uint64(m.ID),
		TeamID:     uint64(m.TeamID),
		StatusCode: uint64(res.Response.StatusCode),
		Method:     m.Settings.Method,
		URL:        m.Settings.URL,
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

func triggerIncident(m *entities.Monitor, hr *http.Result, failed *[]entities.MonitorAssertion) error {
	return nil // TODO: implement
}
