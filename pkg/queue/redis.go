package queue

import (
	"bytes"
	"context"
	"encoding/gob"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisQueue struct {
	client   *redis.Client
	runEvery time.Duration
}

func NewRedisQueue(client *redis.Client, runEvery time.Duration) Driver {
	return redisQueue{
		client:   client,
		runEvery: runEvery,
	}
}

func (q redisQueue) EnQueue(ctx context.Context, data Payload) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return err
	}
	return q.client.RPush(ctx, "job_queue", data).Err()
}

func (q redisQueue) DeQueue(ctx context.Context) (<-chan Payload, <-chan error) {
	outCh := make(chan Payload)
	errCh := make(chan error)
	var payload Payload

	go func() {
		tick := time.NewTicker(q.runEvery)
		defer func() {
			defer close(outCh)
			defer close(errCh)
			tick.Stop()
		}()
		for {
			select {
			case <-ctx.Done():
				return

			case <-tick.C:
				data, err := q.client.LPop(ctx, "job_queue").Result()
				if err != nil {
					errCh <- err
					continue
				}
				err = gob.NewDecoder(strings.NewReader(data)).Decode(&payload)
				if err != nil {
					errCh <- err
					continue
				}
				outCh <- payload
			}
		}
	}()

	return outCh, errCh
}
