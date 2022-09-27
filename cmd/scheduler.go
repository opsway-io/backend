package cmd

import (
	"context"

	asynqClient "github.com/opsway-io/backend/internal/connectors/asynq"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/monitor"
	schedule "github.com/opsway-io/backend/internal/schedule"
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
	scheduleService := schedule.NewAsynqSchedule(asynqClient.NewScheduler(ctx, conf.Asynq), nil)

	db, err := postgres.NewClient(ctx, conf.Postgres)
	if err != nil {
		l.WithError(err).Fatal("Failed to create Postgres client")
	}

	monitorService := monitor.NewService(db)
	monitors, err := monitorService.GetMonitors(ctx)

	for _, monitor := range *monitors {
		_, err = scheduleService.Add(ctx, monitor.Settings.Frequency, scheduler.ProbeTask, scheduler.TaskPayload{ID: monitor.Settings.MonitorID, Payload: map[string]string{"URL": monitor.Settings.URL}})
		if err != nil {
			l.Fatal(err)
		}
	}

	if err := scheduleService.Scheduler.Run(); err != nil {
		logrus.Fatalf("could not run server: %v", err)
	}
}
