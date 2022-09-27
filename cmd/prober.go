package cmd

import (
	"context"

	"github.com/hibiken/asynq"

	asynqClient "github.com/opsway-io/backend/internal/connectors/asynq"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/probes"
	scheduler "github.com/opsway-io/backend/internal/schedule"

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
	scheduleService := scheduler.New(nil, asynqClient.NewServer(ctx, conf.Asynq))

	// create task handlers
	handlers := map[scheduler.TaskType]asynq.HandlerFunc{}
	handlers[scheduler.ProbeTask] = scheduler.HandleTask(probeResultService)

	scheduleService.Consume(ctx, handlers)
}

// func consume(ctx context.Context, scheduler scheduler.Schedule, rs result.Service, stream string, group string) {
// 	uniqueID := xid.New().String()

// 	for {
// 		entries, err := scheduler.Consume(ctx, stream, group, uniqueID)
// 		if err != nil {
// 			logrus.WithError(err).Fatal("failed to get stream result")
// 		}

// 		for i := 0; i < len(entries[0].Messages); i++ {
// 			url := fmt.Sprint(entries[0].Messages[i].Values["url"])
// 			orgId := fmt.Sprint(entries[0].Messages[i].Values["id"])

// 			res, err := httpProbe.Probe(http.MethodGet, "http://"+url, nil, nil, time.Second*5)
// 			if err != nil {
// 				logrus.WithError(err).Fatal("error probing url")
// 			}

// 			err = scheduler.Ack(ctx, entries[0].Stream, group, entries[0].Messages[i].ID)
// 			if err != nil {
// 				logrus.WithError(err).Fatal("Error ack message")
// 			}

// 			rs.WriteResult(url, orgId, res)
// 		}
// 	}
// }
