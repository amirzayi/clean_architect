package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/amirzayi/clean_architect/pkg/queue"
	"github.com/redis/go-redis/v9"
)

func NewRedisScheduler(client *redis.Client) Driver {
	return redisScheduler{client: client}
}

type redisScheduler struct {
	client   *redis.Client
	runEvery time.Duration
}

func (r redisScheduler) ScheduleTask(ctx context.Context, scheduleAt time.Time, payload queue.Payload) error {
	pipe := r.client.Pipeline()
	pipe.ZAdd(ctx, "scheduler", redis.Z{
		Score:  float64(scheduleAt.Unix()),
		Member: payload.Title,
	})
	pipe.Set(ctx, fmt.Sprintf("task_%s_%d",payload.Title,scheduleAt.Unix()), nil, 0)
	_, err := pipe.Exec(ctx)
	return err
}

func (r redisScheduler) Start(ctx context.Context) <-chan error {
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		t := time.NewTicker(r.runEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return

			case <-time.NewTicker(time.Second).C:
				data, err := r.client.ZRangeByScore(ctx, "scheduler", &redis.ZRangeBy{
					Min: "-inf",
					Max: strconv.FormatInt(time.Now().Unix(), 10),
				}).Result()
				if err != nil {
					errCh <- err
					continue
				}
				for _, d := range data {
					fmt.Println("executing", d)
					if err = r.client.ZRem(ctx, "scheduler", d).Err(); err != nil {
						errCh <- err
					}

				}
			}
		}
	}()
	return errCh
}
