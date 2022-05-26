package redis

import (
	"fmt"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

func NewConnection() (*redis.Client, error) {
	conf, err := loadEnvConfig()
	if err != nil {
		return nil, err
	}
	rc := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", conf.Host, conf.Port),
	})
	_, err = rc.Ping().Result()
	if err != nil {
		logrus.WithError(err).Fatal("Unable to connect to Redis", err)
	}
	return rc, nil
}
