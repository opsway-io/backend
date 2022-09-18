package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

type Schedule interface {
	Add(ctx context.Context, interval time.Duration, data map[string]interface{}) (id string, err error)
	CreateConsumer(ctx context.Context, stream string, interval time.Duration) (err error)
	Ack(ctx context.Context, stream string, consumersGroup string, id string) (err error)
	TriggerConsumer(ctx context.Context, id string, data interface{}) (err error)
	Remove(ctx context.Context, id string) (err error)
	TriggerSpecific(ctx context.Context, stream string, group string, id string) (err error)
	Consume(ctx context.Context, stream string, consumersGroup string, id string) (msgs []redis.XStream, err error)
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

func (rs *RedisSchedule) CreateConsumer(ctx context.Context, stream string, interval time.Duration) (err error) {
	consumerName := fmt.Sprintf("%s-consumer-%s", stream, strconv.Itoa(int(interval.Seconds())))
	err = rs.client.XGroupCreate(stream, consumerName, "0").Err()
	if err != nil {
		return err
	}

	setIdCmd := fmt.Sprintf("redis.call('%s', '%s', '%s', '%s', '%s')", rs.client.XGroupSetID(stream, consumerName, "0").Args()...)

	return rs.client.Do("KEYDB.CRON", consumerName, "REPEAT", interval, setIdCmd).Err()
}

func (rs *RedisSchedule) Ack(ctx context.Context, stream string, consumersGroup string, id string) (err error) {
	return rs.client.XAck(stream, consumersGroup, id).Err()
}

func (rs *RedisSchedule) TriggerConsumer(ctx context.Context, id string, data interface{}) (err error) {
	return rs.client.Set(id, data, 0).Err()
}

func (rs *RedisSchedule) Remove(ctx context.Context, id string) (err error) {
	return rs.client.Del(id).Err()
}

func (rs *RedisSchedule) TriggerSpecific(ctx context.Context, stream string, group string, id string) (err error) {
	return rs.client.XGroupSetID(stream, group, id).Err()
}

func (rs *RedisSchedule) Consume(ctx context.Context, stream string, consumersGroup string, id string) (msgs []redis.XStream, err error) {
	readGroupArgs := redis.XReadGroupArgs{
		Group:    consumersGroup,
		Consumer: id,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    -1,
		NoAck:    false,
	}

	return rs.client.XReadGroup(&readGroupArgs).Result()
}
