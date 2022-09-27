package cmd

import (
	"context"

	"github.com/hibiken/asynq"

	asynqClient "github.com/opsway-io/backend/internal/connectors/asynq"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/probes"
	schedule "github.com/opsway-io/backend/internal/schedule"

	"github.com/spf13/cobra"
)

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
	conf, err := loadConfig()
	if err != nil {
		panic(err)
	}

	l := getLogger(conf.Log)

	ctx := context.Background()

	db, err := clickhouse.NewClient(ctx, conf.Clickhouse)
	if err != nil {
		l.WithError(err).Fatal("Failed to create clickhouse")
	}

	db.AutoMigrate(
		entities.ProbeResult{},
	)

	probeResultService := probes.NewService(db)

	l.WithField("addr", conf.Asynq.Addr).Info("connecting to asynq")
	scheduleService := schedule.NewAsynqSchedule(nil, asynqClient.NewServer(ctx, conf.Asynq))

	// create task handlers
	handlers := map[schedule.TaskType]asynq.HandlerFunc{}
	handlers[schedule.ProbeTask] = schedule.HandleTask(probeResultService)

	scheduleService.Consume(ctx, handlers)
}
