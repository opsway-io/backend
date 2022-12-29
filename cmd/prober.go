package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	connectorRedis "github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

type ProberConfig struct {
	Concurrency int `mapstructure:"concurrency" default:"1"`
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

	l.Info("Waiting for tasks...")

	if err := schedule.On(ctx, func(ctx context.Context, monitor *entities.Monitor) {
		handleTask(ctx, l, monitor, httpResultService)
	}); err != nil {
		l.WithError(err).Fatal("failed to start schedule")
	}

	l.Info("Goodbye!")
}

func handleTask(ctx context.Context, l *logrus.Logger, m *entities.Monitor, c check.Service) {
	res, err := http.Probe(
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

	l.WithFields(logrus.Fields{
		"monitor_id": m.ID,
		"status":     res.Response.StatusCode,
		"total":      fmt.Sprintf("%v", res.Timing.Phases.Total),
	}).Info("probe successful")

	timings, err := json.Marshal(res.Timing.Phases)
	if err != nil {
		l.WithError(err).Error("failed to marshal timings")

		return
	}

	tls, err := json.Marshal(res.TLS)
	if err != nil {
		l.WithError(err).Error("failed to marshal tls")

		return
	}

	result := check.Check{
		StatusCode: uint64(res.Response.StatusCode),
		Timing:     string(timings),
		TLS:        string(tls),
		MonitorID:  m.ID,
	}

	err = c.Create(ctx, &result)
	if err != nil {
		l.WithError(err).Error("failed add result to clickhouse")
	}

}
