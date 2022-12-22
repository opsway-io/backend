package cmd

import (
	"context"
	"fmt"
	"time"

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

	redisClient, err := connectorRedis.NewClient(ctx, conf.Redis)
	if err != nil {
		l.WithError(err).Fatal("failed to connect to redis")
	}

	l.WithFields(logrus.Fields{
		"host": conf.Redis.Host,
		"port": conf.Redis.Port,
		"db":   conf.Redis.DB,
	}).Info("Connected to redis")

	schedule := monitor.NewSchedule(redisClient)

	l.Info("Waiting for tasks...")

	if err := schedule.On(ctx, func(ctx context.Context, monitor *entities.Monitor) {
		handleTask(ctx, l, monitor)
	}); err != nil {
		l.WithError(err).Fatal("failed to start schedule")
	}

	l.Info("Goodbye!")
}

func handleTask(ctx context.Context, l *logrus.Logger, m *entities.Monitor) {
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
		"status": res.Response.StatusCode,
		"total":  fmt.Sprintf("%v", res.Timing.Phases.Total),
	}).Info("probe successful")

	// TODO: save result
}
