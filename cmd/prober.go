package cmd

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"

	asynqClient "github.com/opsway-io/backend/internal/connectors/asynq"

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

	// ctx := context.Background()

	// influxc, err := influxClient.NewClient(ctx, influxClient.Config{ServerURL: "http://localhost:8086", Token: "aIlugity6YsoMsmHybWgAWy37pVtP-06qmE2vNLGVmni8k33G_VdzIDaw7nUa9Pnp9UI0AdHGDJeHfkuNI7o_Q=="})
	// if err != nil {
	// 	panic(err)
	// }

	// resultService, err := result.NewService(influxc, "1", "http")
	// if err != nil {
	// 	panic(err)
	// }

	l.WithField("addr", conf.Asynq.Addr).Info("connecting to asynq")
	asynqClient.NewHandler("probe:http", probe, conf.Asynq)

}

type ProbeTaskPayload struct {
	Url string
	Id  string
}

func probe(ctx context.Context, t *asynq.Task) error {
	var p ProbeTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log.Printf(" [*] Probe %s", p.Url)
	return nil
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
