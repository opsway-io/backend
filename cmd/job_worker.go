package cmd

import (
	"fmt"

	"github.com/go-redis/redis"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command.
var serveCmd = &cobra.Command{
	Use: "sub",
	Run: serve,
}

func init() { //nolint:gochecknoinits
	rootCmd.AddCommand(serveCmd)
}

func serve(cmd *cobra.Command, args []string) {
	rc := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", "keydb", "6379"),
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
			// logrus.WithError(err).Fatal("fuck")

			if c != 0 {
				fmt.Println("count: %d", c)
				c = 0
			}

			continue
		}

		for i := 0; i < len(entries[0].Messages); i++ {
			err := handleNewTicket(rc, entries[0].Messages[i])
			if err != nil {
				logrus.WithError(err).Fatal(err)
			}
			c++
		}
	}
}

func handleNewTicket(rc *redis.Client, msg redis.XMessage) error {
	messageID := msg.ID
	values := msg.Values

	ticketID := fmt.Sprintf("%v", values["ticketID"])
	ticketData := fmt.Sprintf("%v", values["ticketData"])

	logrus.Printf("Handling new ticket id : %s data %s\n", ticketID, ticketData)

	return rc.XAck(subject, consumersGroup, messageID).Err()
}
