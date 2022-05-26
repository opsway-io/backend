package main

import (
	"monitor/internal/influxdb"

	"github.com/sirupsen/logrus"
)

const (
	subject        = "tickets"
	consumersGroup = "tickets-consumer-group"
)

func main() {
	// cmd.Execute()

	client, err := influxdb.NewConnection()
	if err != nil {
		logrus.Fatal(err)
	}

	repo, err = influxdb.NewRepository(client, "test", "test")
	if err != nil {
		logrus.Fatal(err)
	}
}
