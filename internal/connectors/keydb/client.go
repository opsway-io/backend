package keydb

import (
	"context"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

type Config struct {
	Addr string `required:"true"`
}

func NewClient(ctx context.Context, conf Config) (*redis.Client, error) {
	rc := redis.NewClient(&redis.Options{
		Addr: conf.Addr,
	})

	if _, err := rc.WithContext(ctx).Ping().Result(); err != nil {
		return nil, errors.Wrap(err, "unable to ping keydb")
	}

	return rc, nil
}
