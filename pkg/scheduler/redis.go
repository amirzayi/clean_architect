package scheduler

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisStorage struct {
	client *redis.Client
}

func NewRedisStorage(client *redis.Client) Storage {
	return redisStorage{client: client}
}

func (s redisStorage) Store(ctx context.Context, taskName string, scheduleAt time.Time, payload []byte) error {
	pipe := s.client.Pipeline()
	id := fmt.Sprintf("task_%s_%d_%d", taskName, scheduleAt.Unix(), rand.IntN(100))
	pipe.ZAdd(ctx, "scheduler", redis.Z{
		Score:  float64(scheduleAt.Unix()),
		Member: id,
	})
	pipe.Set(ctx, id, payload, 0)
	_, err := pipe.Exec(ctx)
	return err
}

func (s redisStorage) Retrieve(ctx context.Context) (taskID string, taskName string, data []byte, err error) {
	taskIDs, err := s.client.ZRangeByScore(ctx, "scheduler", &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(time.Now().Unix(), 10),
		Count: 1,
	}).Result()
	if err != nil {
		return
	}
	if len(taskIDs) < 1 {
		return
	}

	taskID = taskIDs[0]
	taskName = strings.Split(strings.TrimLeft(taskID, "task_"), "_")[0]
	data, err = s.client.Get(ctx, taskID).Bytes()
	if err != nil {
		return
	}
	err = s.client.ZRem(ctx, "scheduler", taskID).Err()
	return
}

func (s redisStorage) Failure(ctx context.Context, taskID string) error {
	return s.client.ZAdd(ctx, "scheduler-retry", redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: taskID,
	}).Err()
}

func (s redisStorage) Done(ctx context.Context, taskID string) error {
	return s.client.Del(ctx, taskID).Err()
}
