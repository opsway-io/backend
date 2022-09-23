package cmd

import (
	"context"
	"time"

	asynqClient "github.com/opsway-io/backend/internal/connectors/asynq"
	scheduler "github.com/opsway-io/backend/internal/schedule"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

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
	conf, err := loadConfig()
	if err != nil {
		panic(err)
	}

	l := getLogger(conf.Log)

	ctx := context.Background()

	l.WithField("addr", conf.Asynq.Addr).Info("connecting to asynq")
	scheduleService := scheduler.New(asynqClient.NewScheduler(ctx, conf.Asynq), nil)

	scheduleService.Add(ctx, time.Second*10, scheduler.ProbeTask, scheduler.TaskPayload{ID: 1, Payload: map[string]string{"URL": "opsway.io"}})
	scheduleService.Add(ctx, time.Second*20, scheduler.ProbeTask, scheduler.TaskPayload{ID: 2, Payload: map[string]string{"URL": "google.com"}})

	if err := scheduleService.Scheduler.Run(); err != nil {
		logrus.Fatalf("could not run server: %v", err)
	}

}
