package scheduler

import (
	"context"
	"time"

	"github.com/amirzayi/clean_architect/pkg/queue"
)

type Driver interface {
	ScheduleTask(ctx context.Context, task string, scheduleAt time.Time, payload queue.Payload) error
	Start(ctx context.Context) <-chan error
	RegisterExecutor(task string, executor JobExecutor)
}

type JobExecutor interface {
	Execute(ctx context.Context, payload queue.Payload)error
}
