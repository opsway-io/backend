package influxdb

import (
	"context"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/pkg/errors"
)

type Config struct {
	ServerURL string `required:"true"`
	Token     string `required:"true"`
}

func NewClient(ctx context.Context, conf Config) (influxdb2.Client, error) {
	client := influxdb2.NewClient(conf.ServerURL, conf.Token)

	ok, err := client.Ping(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to ping influxdb")
	}

	if !ok {
		return nil, errors.New("influxdb not running")
	}

	return client, nil
}
