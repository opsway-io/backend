package cmd

import (
	"context"
	"net/http"
	"time"

	"github.com/opsway-io/backend/internal/connectors/keydb"
	httpProbe "github.com/opsway-io/backend/internal/probes/http"
	result "github.com/opsway-io/backend/internal/results"
	scheduler "github.com/opsway-io/backend/internal/schedule"

	influxClient "github.com/opsway-io/backend/internal/connectors/influxdb"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
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

	l.WithField("addr", conf.KeyDB.Addr).Info("connecting to keydb")
	redisClient, err := keydb.NewClient(ctx, conf.KeyDB)
	if err != nil {
		panic(err)
	}

	redisScheduler := scheduler.New(redisClient)

	influxc, err := influxClient.NewClient(ctx, influxClient.Config{ServerURL: "http://localhost:8086", Token: "aIlugity6YsoMsmHybWgAWy37pVtP-06qmE2vNLGVmni8k33G_VdzIDaw7nUa9Pnp9UI0AdHGDJeHfkuNI7o_Q=="})
	if err != nil {
		panic(err)
	}

	resultService, err := result.NewService(influxc, "1", "http")
	if err != nil {
		panic(err)
	}

	subject := "stream-1"
	group := "stream-1"

	// err = redisScheduler.CreateGroup(ctx, subject, group)
	// if err != nil {
	// 	panic(err)
	// }

	consume(ctx, redisScheduler, resultService, subject, group)
}

func consume(ctx context.Context, scheduler scheduler.Schedule, rs result.Service, subject string, group string) {
	uniqueID := xid.New().String()

	for {
		entries, err := scheduler.Consume(ctx, subject, group, uniqueID)
		if err != nil {
			logrus.WithError(err).Fatal("failed to get stream result")
		}

		for i := 0; i < len(entries[0].Messages); i++ {
			res, err := httpProbe.Probe(http.MethodGet, "https://opsway.io", nil, nil, time.Second*5)
			if err != nil {
				logrus.WithError(err).Fatal(err)
			}

			err = scheduler.Ack(ctx, entries[0].Stream, group, entries[0].Messages[i].ID)
			if err != nil {
				logrus.WithError(err).Fatal(err)
			}

			rs.WriteResult("https://opsway.io", "opsway", res)
		}
	}
}
