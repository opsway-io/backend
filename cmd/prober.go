package cmd

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	connectorRedis "github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/boomerang"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

type ProberConfig struct {
	Concurrency int `mapstructure:"concurrency"`
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
	ctx, cancel := context.WithCancel(cmd.Context())
	var wg *sync.WaitGroup

	conf, err := loadConfig()
	if err != nil {
		panic(err)
	}

	l := getLogger(conf.Log)

	// Connect to redis

	redisClient, err := connectorRedis.NewClient(ctx, conf.Redis)
	if err != nil {
		l.WithError(err).Fatal("failed to connect to redis")
	}

	l.WithFields(logrus.Fields{
		"host": conf.Redis.Host,
		"port": conf.Redis.Port,
		"db":   conf.Redis.DB,
	}).Info("Connected to redis")

	// Connect to clickhouse and initialize probes services

	/*
		db, err := clickhouse.NewClient(ctx, conf.Clickhouse)
		if err != nil {
			l.WithError(err).Fatal("Failed to connect to clickhouse")
		}

		l.Info("Connected to clickhouse")

		probeResultService := probes.NewService(db)
	*/

	// Register consumers

	schedule := boomerang.NewSchedule(redisClient)

	for i := 0; i < conf.Prober.Concurrency; i++ {
		go func() {
			wg.Add(1)
			defer wg.Done()

			schedule.Consume(
				ctx,
				"probe:http",
				[]string{"eu-central-1"},
				handleHttpProbe,
			)
		}()
	}

	// Wait for interrupt signal to gracefully shutdown the application

	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	l.Info("Shutting down...")

	cancel()

	wg.Wait()
}

func handleHttpProbe(ctx context.Context, task boomerang.Task) error {
	// TODO: implement

	return nil
}
