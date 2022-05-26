package main

import (
	"github.com/opsway-io/backend/internal/influxdb"

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

	_, err = influxdb.NewRepository(client, "test", "test")
	if err != nil {
		logrus.Fatal(err)
	}
}
