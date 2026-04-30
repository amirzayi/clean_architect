package bus

import (
	"bytes"
	"context"
	"encoding/gob"
)

type Driver interface {
	Publish(subject string, content []byte) error
	Subscribe(subject string) (dataCh <-chan []byte, errCh <-chan error, err error)
}

type EventBus[T any] interface {
	Publish(ctx context.Context, subject string, content T) error
	Subscribe(queue string) (contentCh <-chan T, errCh <-chan error, err error)
}

type typedEventBus[T any] struct {
	drv Driver
}

func New[T any](drv Driver) EventBus[T] {
	return &typedEventBus[T]{drv: drv}
}

func (b *typedEventBus[T]) Publish(ctx context.Context, subject string, content T) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(content); err != nil {
		return err
	}
	return b.drv.Publish(subject, buf.Bytes())
}

func (b *typedEventBus[T]) Subscribe(subject string) (<-chan T, <-chan error, error) {
	data, busErrs, err := b.drv.Subscribe(subject)
	if err != nil {
		return nil, nil, err
	}

	out := make(chan T)
	errCh := make(chan error)

	go func() {
		for err := range busErrs {
			errCh <- err
		}
	}()

	go func(byteCh <-chan []byte, errCh chan<- error) {
		var v T
		for b := range byteCh {
			if err = gob.NewDecoder(bytes.NewReader(b)).Decode(&v); err != nil {
				errCh <- err
				continue
			}
			out <- v
		}
	}(data, errCh)

	return out, errCh, nil
}
