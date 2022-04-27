package scheduler

import (
	"context"
	"errors"

	"github.com/go-redis/redis"
)

type ConsumerFunc func(id string, data interface{})

type Schedule interface {
	Add(cron string, data interface{}, labels []string) (id string, err error)
	Update(id string, data interface{}) (err error)
	Remove(id string) (err error)
	Run(id string) (err error)
	Consume(ctx context.Context, labels []string, f ConsumerFunc) (err error)
}

type RedisSchedule struct {
	client *redis.Client
}

func New(client *redis.Client) Schedule {
	return &RedisSchedule{client: client}
}

func (rs *RedisSchedule) Add(cron string, data interface{}, labels []string) (id string, err error) {
	return "", errors.New("not implemented")
}

func (rs *RedisSchedule) Update(id string, data interface{}) (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Remove(id string) (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Run(id string) (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Consume(ctx context.Context, labels []string, f ConsumerFunc) (err error) {
	return errors.New("not implemented")
}
