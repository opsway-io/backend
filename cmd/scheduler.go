package cmd

import (
	"context"
	"time"

	asynqClient "github.com/opsway-io/backend/internal/connectors/asynq"
	schedule "github.com/opsway-io/backend/internal/schedule"
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
	scheduleService := schedule.NewAsynqSchedule(asynqClient.NewScheduler(ctx, conf.Asynq), nil)

	_, err = scheduleService.Add(ctx, time.Second*10, schedule.ProbeTask, schedule.TaskPayload{ID: 1, Payload: map[string]string{"URL": "opsway.io"}})
	if err != nil {
		l.Fatal(err)
	}
	_, err = scheduleService.Add(ctx, time.Second*20, schedule.ProbeTask, schedule.TaskPayload{ID: 2, Payload: map[string]string{"URL": "google.com"}})
	if err != nil {
		l.Fatal(err)
	}

	if err := scheduleService.Scheduler.Run(); err != nil {
		logrus.Fatalf("could not run server: %v", err)
	}
}
