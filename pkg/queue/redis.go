package queue

import (
	"bytes"
	"context"
	"encoding/gob"
	"strings"

	"github.com/redis/go-redis/v9"
)

type redisQueue struct {
	client *redis.Client
}

func NewRedisQueue(client *redis.Client) Driver {
	return redisQueue{client: client}
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
		for {
			data, err := q.client.BLPop(ctx, 0, "job_queue").Result()
			if err != nil {
				errCh <- err
				continue
			}
			if len(data) == 0 {
				continue
			}
			err = gob.NewDecoder(strings.NewReader(data[1])).Decode(&payload)
			if err != nil {
				errCh <- err
				continue
			}
			outCh <- payload
		}
	}()

	return outCh, errCh
}
