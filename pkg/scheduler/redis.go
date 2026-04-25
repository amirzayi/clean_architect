package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/amirzayi/clean_architect/pkg/queue"
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

var ErrExecutorNotDefined = errors.New("executor not defined for this task")

func (r redisScheduler) ScheduleTask(ctx context.Context, task string, scheduleAt time.Time, payload queue.Payload) error {
	if _, exists := r.executors[task]; !exists {
		return ErrExecutorNotDefined
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	pipe := r.client.Pipeline()
	key := fmt.Sprintf("task_%s_%d_%d", task, scheduleAt.Unix(), rand.IntN(100))
	pipe.ZAdd(ctx, "scheduler", redis.Z{
		Score:  float64(scheduleAt.Unix()),
		Member: key,
	})
	pipe.Set(ctx, key, data, 0)
	_, err = pipe.Exec(ctx)
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
					if err = json.Unmarshal(data, &payload); err != nil {
						errCh <- err
						continue
					}
					r.sem <- struct{}{}
					go func() {
						defer func() {
							if err = r.client.ZRem(ctx, "scheduler", task).Err(); err != nil {
								errCh <- err
							}
							<-r.sem
						}()
						if err = executor(ctx, payload); err != nil {
							err = r.client.ZAdd(ctx, "scheduler", redis.Z{
								Score:  float64(time.Now().Unix()),
								Member: task,
							}).Err()
							if err != nil {
								errCh <- err
							}
							errCh <- fmt.Errorf("failed to execute task, %w", err)
							return
						}
						if err = r.client.Del(ctx, task).Err(); err != nil {
							errCh <- err
						}
					}()
				}
			}
		}
	}()
	return errCh
}
