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

type SchedulerConfig struct {
	Concurrency int `mapstructure:"concurrency" default:"1"`
}

//nolint:gochecknoglobals
var schedulerCmd = &cobra.Command{
	Use: "scheduler",
	Run: runScheduler,
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(schedulerCmd)
}

func runScheduler(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(cmd.Context())
	var wg sync.WaitGroup

	conf, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
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

	// Start schedulers

	schedule := boomerang.NewSchedule(redisClient)

	l.WithFields(logrus.Fields{
		"concurrency": conf.Scheduler.Concurrency,
	}).Infof("Starting scheduler with %d workers", conf.Scheduler.Concurrency)

	for i := 0; i < conf.Scheduler.Concurrency; i++ {
		go func() {
			wg.Add(1)
			defer wg.Done()

			if err := schedule.Run(ctx); err != nil {
				l.WithError(err).Fatal("failed to run schedule")
			}
		}()
	}

	l.WithFields(logrus.Fields{
		"concurrency": conf.Scheduler.Concurrency,
	}).Info("Scheduler(s) started")

	// Wait for interrupt signal to gracefully shutdown the application

	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	l.Info("Shutting down...")

	cancel()

	wg.Wait()
}
