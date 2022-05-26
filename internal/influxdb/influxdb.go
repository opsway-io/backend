package influxdb

import (
	"fmt"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

func NewConnection() (influxdb2.Client, error) {
	conf, err := loadEnvConfig()
	if err != nil {
		return nil, err
	}

	host := fmt.Sprintf("http://%s:%d", conf.Host, conf.Port)

	client := influxdb2.NewClient(host, conf.Token)

	return client, nil
}
