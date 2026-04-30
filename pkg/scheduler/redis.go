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

type redisScheduler struct {
	client    *redis.Client
	runEvery  time.Duration
	executors map[string]JobExecutor
	sem       chan struct{}
}

func NewRedisScheduler(client *redis.Client, runEvery time.Duration, concurrency int) Driver {
	return redisScheduler{
		client:    client,
		runEvery:  runEvery,
		executors: make(map[string]JobExecutor),
		sem:       make(chan struct{}, concurrency),
	}
}

func (r redisScheduler) RegisterExecutor(task string, executor JobExecutor) {
	r.executors[task] = executor
}

func (r redisScheduler) ScheduleTask(ctx context.Context, task string, scheduleAt time.Time, payload []byte) error {
	if _, exists := r.executors[task]; !exists {
		return ErrExecutorNotDefined
	}
	pipe := r.client.Pipeline()
	key := fmt.Sprintf("task_%s_%d_%d", task, scheduleAt.Unix(), rand.IntN(100))
	pipe.ZAdd(ctx, "scheduler", redis.Z{
		Score:  float64(scheduleAt.Unix()),
		Member: key,
	})
	pipe.Set(ctx, key, payload, 0)
	_, err := pipe.Exec(ctx)
	return err
}

func (r redisScheduler) Start(ctx context.Context) <-chan error {
	errCh := make(chan error)
	go func() {
		t := time.NewTicker(r.runEvery)
		defer func() {
			close(r.sem)
			close(errCh)
			t.Stop()
		}()
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return

			case <-time.NewTicker(time.Second).C:
				r.sem <- struct{}{}
				tasks, err := r.client.ZRangeByScore(ctx, "scheduler", &redis.ZRangeBy{
					Min: "-inf",
					Max: strconv.FormatInt(time.Now().Unix(), 10),
				}).Result()
				if err != nil {
					errCh <- err
					continue
				}
				for _, task := range tasks {
					taskName := strings.Split(strings.TrimLeft(task, "task_"), "_")[0]
					executor, exists := r.executors[taskName]
					if !exists {
						errCh <- ErrExecutorNotDefined
						continue
					}
					data, err := r.client.Get(ctx, task).Bytes()
					if err != nil {
						errCh <- err
						continue
					}
					go func() {
						defer func() {
							<-r.sem
						}()
						if err = r.client.ZRem(ctx, "scheduler", task).Err(); err != nil {
							errCh <- err
						}
						if err = executor(ctx, data); err != nil {
							errCh <- fmt.Errorf("failed to execute task, %w", err)
							err = r.client.ZAdd(ctx, "scheduler-retry", redis.Z{
								Score:  float64(time.Now().Unix()),
								Member: task,
							}).Err()
							if err != nil {
								errCh <- fmt.Errorf("failed to add retry task, %w", err)
							}
							return
						}
						if err = r.client.Del(ctx, task).Err(); err != nil {
							errCh <- fmt.Errorf("failed to remove task, %w", err)
						}
					}()
				}
			}
		}
	}()
	return errCh
}
