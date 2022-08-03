package cmd

import (
	"context"
	"net/http"
	"time"

	"github.com/fatih/structs"
	"github.com/jeremywohl/flatten"
	"github.com/opsway-io/backend/internal/connectors/keydb"
	httpProbe "github.com/opsway-io/backend/internal/probes/http"
	influxRepo "github.com/opsway-io/backend/internal/results/http"

	influxClient "github.com/opsway-io/backend/internal/connectors/influxdb"

	"github.com/go-redis/redis"
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
	client, err := keydb.NewClient(ctx, conf.KeyDB)
	if err != nil {
		panic(err)
	}

	influxc, err := influxClient.NewClient(ctx, influxClient.Config{ServerURL: "http://localhost:8086", Token: "Raz0dd73B0aprtu-GKaaHHgobLcbAzZD1K3fbLLG7HVHw1zRWN2ljFbh0bd-2_4oxjyii3SLt6t01Ev3kdd8QA=="})
	if err != nil {
		panic(err)
	}

	influxr, err := influxRepo.NewRepository(influxc, "123", "123")
	if err != nil {
		panic(err)
	}

	consume(client, influxr)
}

func consume(rc *redis.Client, ir *influxRepo.Repository) {
	uniqueID := xid.New().String()

	readGroupArgs := redis.XReadGroupArgs{
		Group:    "TODO",
		Consumer: uniqueID,
		Streams:  []string{"TODO", ">"},
		Count:    1,
		Block:    -1,
		NoAck:    false,
	}

	for {
		entries, err := rc.XReadGroup(&readGroupArgs).Result()
		if err != nil {
			logrus.WithError(err).Fatal("failed to get stream result")
		}

		for i := 0; i < len(entries[0].Messages); i++ {
			result, err := handleMessage(rc, entries[0].Messages[i])
			if err != nil {
				logrus.WithError(err).Fatal(err)
			}

			writeResult(result)
		}
	}
}

func handleMessage(rc *redis.Client, msg redis.XMessage) (*httpProbe.Result, error) {
	messageID := msg.ID

	// TODO: use real values
	res, err := httpProbe.Probe(http.MethodGet, "https://opsway.io", nil, nil, time.Second*5)
	if err != nil {
		return nil, err
	}

	return res, rc.XAck("TODO", "TODO", messageID).Err()
}

func writeResult(res *httpProbe.Result) {
	m := structs.Map(res)
	m, err := flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		logrus.WithError(err).Fatal(err)
	}

	influxRepo.Repository.Write()

	// TODO: Write it somewhere
}
