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
	Redis       connectorRedis.Config `mapstructure:"redis"`
	Concurrency int                   `mapstructure:"concurrency"`
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
		panic(err)
	}

	// Connect to redis

	redisClient, err := connectorRedis.NewClient(ctx, conf.Scheduler.Redis)
	if err != nil {
		logrus.New().WithError(err).Fatal("failed to connect to redis")
	}

	logrus.WithFields(logrus.Fields{
		"host": conf.Redis.Host,
		"port": conf.Redis.Port,
		"db":   conf.Redis.DB,
	}).Info("Connected to redis")

	// Start schedulers

	schedule := boomerang.NewSchedule(redisClient)

	logrus.WithFields(logrus.Fields{
		"concurrency": conf.Scheduler.Concurrency,
	}).Infof("Starting scheduler with %d workers", conf.Scheduler.Concurrency)

	for i := 0; i < conf.Scheduler.Concurrency; i++ {
		go func() {
			wg.Add(1)
			defer wg.Done()

			if err := schedule.Run(ctx); err != nil {
				logrus.New().WithError(err).Fatal("failed to run schedule")
			}
		}()
	}

	logrus.Info("Scheduler(s) started")

	// Wait for interrupt signal to gracefully shutdown the application

	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	logrus.Info("Shutting down...")

	cancel()

	wg.Wait()
}
