package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/opsway-io/backend/internal/checker"
	"github.com/opsway-io/backend/pkg/utils"

	"github.com/go-redis/redis"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command.
var httpProbeCmd = &cobra.Command{
	Use: "probe",
	Run: httpProbe,
}

func init() { //nolint:gochecknoinits
	rootCmd.AddCommand(httpProbeCmd)
}

func httpProbe(cmd *cobra.Command, args []string) {
	rc := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", "127.0.0.1", "6379"),
	})
	_, err := rc.Ping().Result()
	if err != nil {
		logrus.WithError(err).Fatal("Unable to connect to Redis", err)
	}

	uniqueID := xid.New().String()
	c := 0
	for {
		entries, err := rc.XReadGroup(&redis.XReadGroupArgs{
			Group:    consumersGroup,
			Consumer: uniqueID,
			Streams:  []string{subject, ">"},
			Count:    1,
			Block:    -1,
			NoAck:    false,
		}).Result()
		if err != nil {
			if c != 0 {
				fmt.Println("count: %d", c)
				c = 0
			}

			continue
		}

		for i := 0; i < len(entries[0].Messages); i++ {
			err := handle(rc, entries[0].Messages[i])
			if err != nil {
				logrus.WithError(err).Fatal(err)
			}
			c++
		}
	}
}

func handle(rc *redis.Client, msg redis.XMessage) error {
	messageID := msg.ID

	res, err := checker.APICheck(http.MethodGet, "https://tranberg.tk", nil, nil, time.Second*5)
	if err != nil {
		logrus.WithError(err).Error("HTTP probe failed")
	}

	utils.Print(res)

	return rc.XAck(subject, consumersGroup, messageID).Err()
}
