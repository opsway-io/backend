package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/gammazero/workerpool"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	connectorRedis "github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/opsway-io/backend/internal/probes/multistep"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

type ProberConfig struct {
	Concurrency int `mapstructure:"concurrency" default:"25"`
	// Locations are the probe locations monitors can be scheduled in.
	Locations []string `mapstructure:"locations"`
}

//nolint:gochecknoglobals
var proberCmd = &cobra.Command{
	Use: "prober",
	Run: runProber,
}

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

	db, err := postgres.NewClient(ctx, conf.Postgres)
	if err != nil {
		l.WithError(err).Fatal("Failed to create Postgres client")
	}

	incidentRepository := incident.NewRepository(db)
	incidentService := incident.NewService(incidentRepository, proberIncidentOptions(conf, l, db)...)

	prober := http.NewService(conf.HTTPProbe)
	runner := multistep.NewRunner(prober, multistep.NewTokenSource(nil))

	l.Info("Waiting for tasks...")

	if err := schedule.On(ctx, func(ctx context.Context, monitor *entities.Monitor) {
		wp.Submit(func() {
			handleTask(ctx, l, runner, monitor, httpResultService, incidentService)
		})
	}); err != nil {
		l.WithError(err).Fatal("failed to start schedule")
	}

	l.Info("Shutting down...")

	wp.StopWait()

	l.Info("Goodbye!")
}

func handleTask(ctx context.Context, logger *logrus.Logger, runner *multistep.Runner, m *entities.Monitor, c check.Service, i incident.Service) {
	l := logger.WithFields(logrus.Fields{
		"monitor_id": m.ID,
	})

	run := runner.Run(ctx, m)
	if len(run.Steps) == 0 {
		l.Warn("monitor has no steps, skipping")

		return
	}

	for _, s := range run.Steps {
		sl := l.WithFields(logrus.Fields{
			"step":              s.Index + 1,
			"method":            s.Method,
			"url":               s.URL,
			"assertions_passed": len(s.Passed),
			"assertions_failed": len(s.Failed),
		})

		if s.Result != nil {
			sl = sl.WithFields(logrus.Fields{
				"status":     s.Result.Response.StatusCode,
				"total_time": fmt.Sprintf("%v", s.Result.Timing.Phases.Total),
			})
		}

		if s.Err != nil {
			sl = sl.WithError(s.Err)
		}

		if len(s.TLSErrors) > 0 {
			sl = sl.WithField("tls_errors", s.TLSErrors)
		}

		if len(s.VariableErrors) > 0 {
			sl = sl.WithField("variable_errors", s.VariableErrors)
		}

		if s.OK() {
			sl.Debug("step passed")
		} else {
			sl.Info("step failed")
		}
	}

	if newCheck := mapRunToCheck(m, run); newCheck != nil {
		if err := c.Create(ctx, newCheck); err != nil {
			l.WithError(err).Error("failed add result to clickhouse")
		}
	}

	failed := run.FailedAssertions()

	l = l.WithFields(logrus.Fields{
		"steps":             len(run.Steps),
		"assertions_failed": len(failed),
	})

	if len(failed) > 0 {
		l.Info("some assertions failed, triggering incident")

		if err := triggerIncident(ctx, m, failed, i); err != nil {
			l.WithError(err).Error("failed to trigger incident")
		}
	} else if run.OK() {
		l.Info("all steps passed")
	} else if inc := stepFailureIncident(m, run); inc != nil {
		// The step failed without a failing assertion (e.g. the host is
		// unreachable, a TLS check failed or a variable could not be
		// extracted). Still open an incident so the failure alerts.
		l.Info("step failed without failing assertions, triggering incident")

		if err := i.Upsert(ctx, &[]entities.Incident{*inc}); err != nil {
			l.WithError(err).Error("failed to trigger incident")
		}
	} else {
		l.Warn("monitor failed but has no assertion to attach an incident to")
	}
}

// stepFailureIncident builds an incident for the first failing step of a run
// that failed without any failing assertion. Incidents are keyed by assertion,
// so it is attached to the step's first assertion, falling back to the
// monitor's first assertion. It returns nil when the monitor has none.
func stepFailureIncident(m *entities.Monitor, run *multistep.RunResult) *entities.Incident {
	if run.FailedStep < 0 || run.FailedStep >= len(run.Steps) {
		return nil
	}

	step := &run.Steps[run.FailedStep]

	var anchor uint
	if step.Step != nil {
		for _, a := range step.Step.Assertions {
			if a.ID != 0 {
				anchor = a.ID
				break
			}
		}
	}
	if anchor == 0 {
		for _, a := range m.Assertions {
			if a.ID != 0 {
				anchor = a.ID
				break
			}
		}
	}
	if anchor == 0 {
		return nil
	}

	var title, detail string
	switch {
	case step.Err != nil:
		title, detail = "Request failed", step.Err.Error()
	case len(step.TLSErrors) > 0:
		title, detail = "TLS check failed", strings.Join(step.TLSErrors, "; ")
	case len(step.VariableErrors) > 0:
		title, detail = "Variable extraction failed", strings.Join(step.VariableErrors, "; ")
	default:
		title, detail = "Step failed", ""
	}

	description := fmt.Sprintf("Step %d: %s %s", step.Index+1, step.Method, step.URL)
	if detail != "" {
		description += " — " + detail
	}

	return &entities.Incident{
		MonitorID:          m.ID,
		TeamID:             m.TeamID,
		Title:              title,
		Description:        &description,
		MonitorAssertionID: anchor,
	}
}

// mapRunToCheck records one check per monitor run. Method, URL and status code
// are taken from the failing step (or the last step when all passed) and the
// timings are summed across all executed steps.
func mapRunToCheck(m *entities.Monitor, run *multistep.RunResult) *check.Check {
	var (
		reported *multistep.StepResult
		timing   check.Timing
		tls      *check.TLS
	)

	for idx := range run.Steps {
		s := &run.Steps[idx]
		reported = s

		if s.Result == nil {
			continue
		}

		p := s.Result.Timing.Phases
		timing.DNSLookup += p.DNSLookup
		timing.TCPConnection += p.TCPConnection
		timing.TLSHandshake += p.TLSHandshake
		timing.ServerProcessing += p.ServerProcessing
		timing.ContentTransfer += p.ContentTransfer
		timing.Total += p.Total

		if tls == nil && s.Result.TLS != nil {
			tls = &check.TLS{
				Version:   s.Result.TLS.Version,
				Cipher:    s.Result.TLS.Cipher,
				Issuer:    s.Result.TLS.Certificate.Issuer.Organization,
				Subject:   s.Result.TLS.Certificate.Subject.CommonName,
				NotBefore: s.Result.TLS.Certificate.NotBefore,
				NotAfter:  s.Result.TLS.Certificate.NotAfter,
			}
		}
	}

	if reported == nil {
		return nil
	}

	c := &check.Check{
		MonitorID: uint64(m.ID),
		TeamID:    uint64(m.TeamID),
		Method:    reported.Method,
		URL:       reported.URL,
		Timing:    timing,
		TLS:       tls,
	}

	// A request that could not be performed is recorded with status code 0
	if reported.Result != nil {
		c.StatusCode = uint64(reported.Result.Response.StatusCode)
	}

	return c
}

func triggerIncident(ctx context.Context, m *entities.Monitor, failed []entities.MonitorAssertion, i incident.Service) error {
	incidents := make([]entities.Incident, 0, len(failed))

	for _, assertion := range failed {
		// Assertions of monitors scheduled before persisting have no ID
		if assertion.ID == 0 {
			continue
		}

		title, description := incident.DescribeAssertion(assertion)
		incidents = append(incidents, entities.Incident{
			MonitorID:          m.ID,
			TeamID:             m.TeamID,
			Title:              title,
			Description:        &description,
			MonitorAssertionID: assertion.ID,
		})
	}

	if len(incidents) == 0 {
		return nil
	}

	return i.Upsert(ctx, &incidents)
}
