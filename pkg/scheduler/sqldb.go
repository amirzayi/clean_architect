package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type taskModel struct {
	ID          string
	Name        string
	Payload     string
	Scheduled_At string
}
type sqlScheduler struct {
	db        *sqlx.DB
	runEvery  time.Duration
	executors map[string]JobExecutor
	sem       chan struct{}
}

func NewSQLScheduler(db *sqlx.DB, runEvery time.Duration, concurrency int) Driver {
	return sqlScheduler{
		db:        db,
		runEvery:  runEvery,
		executors: make(map[string]JobExecutor),
		sem:       make(chan struct{}, concurrency),
	}
}

func (s sqlScheduler) ScheduleTask(ctx context.Context, task string, scheduleAt time.Time, payload []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO task_scheduler
	(id,name,payload,scheduled_at)
	VALUES(?,?,?,?)`,
	uuid.New(),	task, string(payload), scheduleAt.Format(time.DateTime))
	if err != nil {
		return err
	}
	return nil
}

func (s sqlScheduler) Start(ctx context.Context) <-chan error {
	errCh := make(chan error)
	go func() {
		t := time.NewTicker(s.runEvery)
		defer func() {
			close(s.sem)
			close(errCh)
			t.Stop()
		}()
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return

			case <-time.NewTicker(time.Second).C:
				s.sem <- struct{}{}
				var task taskModel
				err := s.db.GetContext(ctx, &task, "SELECT * FROM task_scheduler WHERE scheduled_at < '?' LIMIT 1", time.Now().Format(time.DateTime))
				if err != nil {
					if !errors.Is(err, sql.ErrNoRows) {
						errCh <- err
					}
					continue
				}

				executor, exists := s.executors[task.Name]
				if !exists {
					errCh <- ErrExecutorNotDefined
					continue
				}

				go func() {
					defer func() {
						<-s.sem
					}()
					if err = executor(ctx, []byte(task.Payload)); err != nil {
						errCh <- fmt.Errorf("failed to execute task, %w", err)
					}
					_, err = s.db.ExecContext(ctx, "DELETE FROM task_scheduler WHERE id=?", task.ID)
					if err != nil {
						errCh <- err
					}
				}()
			}
		}
	}()
	return errCh
}

func (s sqlScheduler) RegisterExecutor(task string, executor JobExecutor) {
	s.executors[task] = executor
}
