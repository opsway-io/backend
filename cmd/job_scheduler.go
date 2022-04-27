package cmd

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// jobSchedulerCmd represents the serve command.
var jobSchedulerCmd = &cobra.Command{
	Use: "pub",
	Run: runScheduler,
}

func init() { //nolint:gochecknoinits
	rootCmd.AddCommand(jobSchedulerCmd)
}

const (
	subject        = "tickets"
	consumersGroup = "tickets-consumer-group"
)

func runScheduler(cmd *cobra.Command, args []string) {
	rc := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", "127.0.0.1", "6379"),
	})
	_, err := rc.Ping().Result()
	if err != nil {
		logrus.WithError(err).Fatal("Unable to connect to Redis", err)
	}

	if err := rc.XTrim(subject, 0).Err(); err != nil {
		logrus.WithError(err).Fatal("Unable to connect to Redis", err)
	}

	err = rc.XGroupCreateMkStream(subject, consumersGroup, "0").Err()
	if err != nil {
		if !strings.Contains(err.Error(), "Consumer Group name already exists") {
			logrus.WithError(err).Fatal(err)
		}
	}

	for i := 0; i < 10000; i++ {
		err = publishTicketReceivedEvent(rc)
		if err != nil {
			logrus.WithError(err).Fatal(err)
		}
	}

	for {

		time.Sleep(time.Second * 2)

		logrus.Info("resetting group")
		if err := rc.XGroupSetID(subject, consumersGroup, "0").Err(); err != nil {
			logrus.WithError(err).Fatal(err)
		}
	}
}

func publishTicketReceivedEvent(client *redis.Client) error {
	logrus.Info("Publishing event to Redis")
	err := client.XAdd(&redis.XAddArgs{
		Stream:       "tickets",
		MaxLen:       0,
		MaxLenApprox: 0,
		ID:           "",
		Values: map[string]interface{}{
			"whatHappened": string("ticket received"),
			"ticketID":     int(rand.Intn(100000000)),
			"ticketData":   string("some ticket data"),
		},
	}).Err()

	return err
}
