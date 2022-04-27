package scheduler

import (
	"context"
	"errors"

	"github.com/go-redis/redis"
	"github.com/robfig/cron/v3"
)

type ConsumerFunc func(id string, data interface{})

type Schedule interface {
	Add(ctx context.Context, cron *cron.Schedule, data interface{}) (id string, err error)
	Set(ctx context.Context, id string, data interface{}) (err error)
	Remove(ctx context.Context, id string) (err error)
	Run(ctx context.Context, id string) (err error)
	Consume(ctx context.Context, f ConsumerFunc) (err error)
}

type RedisSchedule struct {
	client *redis.Client
}

func New(client *redis.Client) Schedule {
	return &RedisSchedule{client: client}
}

func (rs *RedisSchedule) Add(ctx context.Context, cron *cron.Schedule, data interface{}) (id string, err error) {
	return "", errors.New("not implemented")
}

func (rs *RedisSchedule) Set(ctx context.Context, id string, data interface{}) (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Remove(ctx context.Context, id string) (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Run(ctx context.Context, id string) (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Consume(ctx context.Context, f ConsumerFunc) (err error) {
	return errors.New("not implemented")
}
