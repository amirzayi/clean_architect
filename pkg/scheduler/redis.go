package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/amirzayi/clean_architect/pkg/queue"
	"github.com/redis/go-redis/v9"
)

func NewRedisScheduler(client *redis.Client) Driver {
	return redisScheduler{client: client}
}

type redisScheduler struct {
	client    *redis.Client
	runEvery  time.Duration
	executors map[string]JobExecutor
}

var ShcedulerNotDefinedErr = errors.New("scheduler not defined for this task")

func (r redisScheduler) ScheduleTask(ctx context.Context, scheduleAt time.Time, payload queue.Payload) error {
	if _, exists := r.executors[payload.Title]; !exists {
		return ShcedulerNotDefinedErr
	}
	pipe := r.client.Pipeline()
	key := fmt.Sprintf("task_%s_%d_%d", payload.Title, scheduleAt.Unix(), rand.IntN(100))
	pipe.ZAdd(ctx, "scheduler", redis.Z{
		Score:  float64(scheduleAt.Unix()),
		Member: key,
	})
	pipe.Set(ctx, key, nil, 0)
	_, err := pipe.Exec(ctx)
	return err
}

func (r redisScheduler) Start(ctx context.Context) <-chan error {
	errCh := make(chan error)
	go func() {
		var payload queue.Payload
		defer close(errCh)
		t := time.NewTicker(r.runEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return

			case <-time.NewTicker(time.Second).C:
				tasks, err := r.client.ZRangeByScore(ctx, "scheduler", &redis.ZRangeBy{
					Min: "-inf",
					Max: strconv.FormatInt(time.Now().Unix(), 10),
				}).Result()
				if err != nil {
					errCh <- err
					continue
				}
				for _, task := range tasks {
					data, err := r.client.Get(ctx, task).Bytes()
					if err != nil {
						errCh <- err
						continue
					}
					if err = json.Unmarshal(data, &payload); err != nil {
						errCh <- err
						continue
					}
					executor, exists := r.executors[payload.Title]
					if !exists {
						errCh <- ShcedulerNotDefinedErr
						continue
					}
					executor.Execute(ctx, payload)
					if err = r.client.ZRem(ctx, "scheduler", task).Err(); err != nil {
						errCh <- err
					}
				}
			}
		}
	}()
	return errCh
}
