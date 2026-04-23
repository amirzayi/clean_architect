package scheduler

import (
	"context"

	"github.com/amirzayi/clean_architect/pkg/queue"
)

type Driver interface {
	// Schedule(ctx context.Context, scheduleAt time.Time) error
	// Start(ctx context.Context) <-chan error
}

type JobExecutor interface {
	Execute(ctx context.Context, payload queue.Payload)
}
