package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

type Config struct {
	User         string
	Password     string
	Host         string `required:"true"`
	Port         uint32 `required:"true"`
	SentinelMode bool   `mapstructure:"sentinel_mode"`
	MasterName   string `mapstructure:"master_name"`
}

func NewClient(ctx context.Context, conf Config) (*redis.Client, error) {
	var cli *redis.Client

	if conf.SentinelMode {
		cli = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    conf.MasterName,
			SentinelAddrs: []string{fmt.Sprintf("%s:%d", conf.Host, conf.Port)},
			MaxRetries:    -1, // Keep retrying...,
			Username:      conf.User,
			Password:      conf.Password,
		})
	} else {
		cli = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", conf.Host, conf.Port),
			Username: conf.User,
			Password: conf.Password,
		})
	}

	if _, err := cli.Ping(ctx).Result(); err != nil {
		return nil, errors.Wrap(err, "Failed to ping redis")
	}

	return cli, nil
}
