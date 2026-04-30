package scheduler

import (
	"context"
	"time"
)

type discard struct {
}

func NewDiscard() Driver {
	return discard{}
}

func (discard) ScheduleTask(_ context.Context, _ string, _ time.Time, _ []byte) error {
	return nil
}

func (discard) Start(_ context.Context) <-chan error {
	ch := make(chan error)
	close(ch)
	return ch
}

func (discard) RegisterExecutor(_ string, _ JobExecutor) {}
