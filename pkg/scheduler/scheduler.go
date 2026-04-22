package scheduler

type Driver interface {
	// Schedule(ctx context.Context, scheduleAt time.Time) error
	// Start(ctx context.Context) <-chan error
}

type Job interface {
	Execute()
}
