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
