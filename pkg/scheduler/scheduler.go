package scheduler

import (
	"context"
	"errors"
	"time"
)

var ErrExecutorNotDefined = errors.New("executor not defined for this task")

type Driver interface {
	ScheduleTask(ctx context.Context, task string, scheduleAt time.Time, payload []byte) error
	Start(ctx context.Context) <-chan error
	RegisterExecutor(task string, executor JobExecutor)
}

type JobExecutor func(ctx context.Context, payload []byte) error

type Storage interface {
	Store(ctx context.Context, taskName string, scheduleAt time.Time, payload []byte) error
	Retrieve(ctx context.Context) (taskIDstring, taskName string, data []byte, err error)
	Failure(ctx context.Context, taskID string) error
	Done(ctx context.Context, taskID string) error
}

type scheduler struct {
	storage   Storage
	runEvery  time.Duration
	executors map[string]JobExecutor
	sem       chan struct{}
}

func NewScheduler(storage Storage, runEvery time.Duration, concurrency int) Driver {
	return scheduler{
		storage:   storage,
		runEvery:  runEvery,
		executors: make(map[string]JobExecutor),
		sem:       make(chan struct{}, concurrency),
	}
}

func (s scheduler) RegisterExecutor(task string, executor JobExecutor) {
	s.executors[task] = executor
}

func (s scheduler) ScheduleTask(ctx context.Context, task string, scheduleAt time.Time, payload []byte) error {
	if _, exists := s.executors[task]; !exists {
		return ErrExecutorNotDefined
	}
	return s.storage.Store(ctx, task, scheduleAt, payload)
}

func (s scheduler) Start(ctx context.Context) <-chan error {
	errCh := make(chan error)
	go func() {
		tick := time.NewTicker(s.runEvery)
		defer func() {
			close(s.sem)
			close(errCh)
			tick.Stop()
		}()
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return

			case <-tick.C:
				taskID, taskName, data, err := s.storage.Retrieve(ctx)
				if err != nil {
					errCh <- err
					continue
				}
				if data == nil {
					continue
				}
				executor, exists := s.executors[taskName]
				if !exists {
					errCh <- ErrExecutorNotDefined
					continue
				}

				s.sem <- struct{}{}
				go func() {
					defer func() {
						<-s.sem
					}()
					if err = executor(ctx, data); err != nil {
						errCh <- err
						if err = s.storage.Failure(ctx, taskID); err != nil {
							errCh <- err
						}
						return
					}
					if err = s.storage.Done(ctx, taskID); err != nil {
						errCh <- err
					}
				}()
			}
		}
	}()
	return errCh
}
