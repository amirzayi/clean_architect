package queue

import (
	"container/list"
	"context"
	"time"
)

type inMemory struct {
	list     *list.List
	runEvery time.Duration
}

func NewInMemory(runEvery time.Duration) Driver {
	return inMemory{
		list:     list.New(),
		runEvery: runEvery,
	}
}

func (m inMemory) EnQueue(_ context.Context, payload Payload) error {
	m.list.PushBack(payload)
	return nil
}

func (m inMemory) DeQueue(ctx context.Context) (<-chan Payload, <-chan error) {
	outCh := make(chan Payload)
	errCh := make(chan error)

	go func() {
		tick := time.NewTicker(m.runEvery)
		defer func() {
			close(outCh)
			close(errCh)
			tick.Stop()
		}()
		for {
			select {
			case <-ctx.Done():
				return

			case <-tick.C:
				if m.list.Len() > 0 {
					outCh <- Payload{Data: m.list.Remove(m.list.Front())}
				}
			}
		}
	}()

	return outCh, errCh
}
