package scheduler

import (
	"context"
	"errors"

	"github.com/go-redis/redis"
	"github.com/robfig/cron/v3"
)

type ConsumerFunc func(id string, data interface{})

type Schedule interface {
	Add(cron *cron.Schedule, data interface{}) (id string, err error)
	Set(id string, data interface{}) (err error)
	Remove(id string) (err error)
	Run(id string) (err error)
	Consume(ctx context.Context, f ConsumerFunc) (err error)
	Enable() (err error)
	Disable() (err error)
}

type RedisSchedule struct {
	client *redis.Client
}

func New(client *redis.Client) Schedule {
	return &RedisSchedule{client: client}
}

func (rs *RedisSchedule) Add(cron *cron.Schedule, data interface{}) (id string, err error) {
	return "", errors.New("not implemented")
}

func (rs *RedisSchedule) Set(id string, data interface{}) (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Remove(id string) (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Run(id string) (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Consume(ctx context.Context, f ConsumerFunc) (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Enable() (err error) {
	return errors.New("not implemented")
}

func (rs *RedisSchedule) Disable() (err error) {
	return errors.New("not implemented")
}
