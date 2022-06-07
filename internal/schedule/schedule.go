package scheduler

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

type Event struct {
	id   string
	data interface{}
}

type Schedule interface {
	Add(ctx context.Context, interval time.Duration, data map[string]interface{}) (id string, err error)
	Set(ctx context.Context, id string, data interface{}) (err error)
	Remove(ctx context.Context, id string) (err error)
	Run(ctx context.Context, id string) (err error)
	Consume(ctx context.Context) (events <-chan *Event, err error)
}

type RedisSchedule struct {
	client *redis.Client
}

func New(client *redis.Client) Schedule {
	return &RedisSchedule{client: client}
}

func (rs *RedisSchedule) Add(ctx context.Context, interval time.Duration, data map[string]interface{}) (id string, err error) {
	logrus.Info("Publishing event to Redis")

	cmd := rs.client.XAdd(&redis.XAddArgs{
		Stream:       "stream-" + strconv.Itoa(int(interval.Seconds())),
		MaxLen:       0,
		MaxLenApprox: 0,
		ID:           "",
		Values:       data,
	})

	return cmd.Result()
}

func (rs *RedisSchedule) Set(ctx context.Context, id string, data interface{}) (err error) {
	return rs.client.Set(id, data, 0).Err()
}

func (rs *RedisSchedule) Remove(ctx context.Context, id string) (err error) {
	return rs.client.Del(id).Err()
}

func (rs *RedisSchedule) Run(ctx context.Context, id string) (err error) {
	rs.client.XRead(&redis.XReadArgs{})
	return rs.client.XGroupSetID("stream-", "consumer-group-", "0").Err()
}

func (rs *RedisSchedule) Consume(ctx context.Context) (events <-chan *Event, err error) {
	return nil, errors.New("not implemented")
}
