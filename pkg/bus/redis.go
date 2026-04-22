package bus

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type redisBroker struct {
	client *redis.Client
}

func NewRedisBroker(client *redis.Client) Driver {
	return redisBroker{client: client}
}

func (r redisBroker) Publish(queue string, data []byte) error {
	return r.client.Publish(context.Background(), queue, data).Err()
}

func (r redisBroker) Subscribe(queue string) (<-chan []byte, <-chan error, error) {
	dataCh := make(chan []byte)
	errCh := make(chan error)
	sub := r.client.Subscribe(context.Background(), queue)
	ch := sub.Channel()
	go func() {
		defer close(dataCh)
		for msg := range ch {
			dataCh <- []byte(msg.Payload)
		}
	}()
	close(errCh)
	return dataCh, errCh, nil
}

func (r redisBroker) Close() error {
	return r.client.Close()
}
