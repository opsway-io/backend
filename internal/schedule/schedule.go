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
	Add(ctx context.Context, stream string, data map[string]interface{}) (id string, err error)
	CreateStream(ctx context.Context, interval time.Duration) (streamName string, err error)
	Ack(ctx context.Context, stream string, consumersGroup string, id string) (err error)
	TriggerConsumerGroup(ctx context.Context, id string, data interface{}) (err error)
	Remove(ctx context.Context, id string) (err error)
	TriggerSpecific(ctx context.Context, stream string, group string, id string) (err error)
	ListConsumerGroups(ctx context.Context, stream string) (consumerGroups []ConsumerGroup, err error)
	Consume(ctx context.Context, stream string, consumersGroup string, id string) (msgs []redis.XStream, err error)
}

type RedisSchedule struct {
	client *redis.Client
}

func New(client *redis.Client) Schedule {
	return &RedisSchedule{client: client}
}

func (rs *RedisSchedule) Add(ctx context.Context, stream string, data map[string]interface{}) (id string, err error) {
	logrus.Info("Publishing event to Redis")

	cmd := rs.client.XAdd(&redis.XAddArgs{
		Stream:       stream,
		MaxLen:       0,
		MaxLenApprox: 0,
		ID:           "",
		Values:       data,
	})

	return cmd.Result()
}

func (rs *RedisSchedule) CreateStream(ctx context.Context, interval time.Duration) (streamName string, err error) {
	streamName = fmt.Sprintf("stream-%s", strconv.Itoa(int(interval.Seconds())))
	consumerName := "consumer-0"

	err = rs.client.XGroupCreateMkStream(streamName, consumerName, "0").Err()
	if err != nil {
		return streamName, err
	}

	setIdCmd := fmt.Sprintf("redis.call('%s', '%s', '%s', '%s', '%s')", rs.client.XGroupSetID(streamName, consumerName, "0").Args()...)

	err = rs.client.Do("KEYDB.CRON", consumerName, "REPEAT", interval.Microseconds(), setIdCmd).Err()

	return streamName, err
}

func (rs *RedisSchedule) Ack(ctx context.Context, stream string, consumersGroup string, id string) (err error) {
	return rs.client.XAck(stream, consumersGroup, id).Err()
}

func (rs *RedisSchedule) TriggerConsumerGroup(ctx context.Context, id string, data interface{}) (err error) {
	return rs.client.Set(id, data, 0).Err()
}

func (rs *RedisSchedule) Remove(ctx context.Context, id string) (err error) {
	return rs.client.Del(id).Err()
}

func (rs *RedisSchedule) TriggerSpecific(ctx context.Context, stream string, group string, id string) (err error) {
	return rs.client.XGroupSetID(stream, group, id).Err()
}

type ConsumerGroup struct {
	Name string
}

func (rs *RedisSchedule) ListConsumerGroups(ctx context.Context, stream string) (consumerGroups []ConsumerGroup, err error) {
	consumerGroups = []ConsumerGroup{}

	result, err := rs.client.Do("XINFO", "GROUPS", stream).Result()
	if err != nil {
		return nil, err
	}

	for _, res := range result.([]interface{}) {
		consumerGroupInfo := res.([]interface{})
		consumerGroups = append(consumerGroups, ConsumerGroup{Name: consumerGroupInfo[1].(string)})
	}

	return consumerGroups, err
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
