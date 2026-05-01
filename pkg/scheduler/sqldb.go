package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type taskModel struct {
	ID           string
	Name         string
	Payload      string
	Scheduled_At string
}

type sqlStorage struct {
	db *sqlx.DB
}

func NewSQLStorage(db *sqlx.DB) Storage {
	return sqlStorage{db: db}
}

func (s sqlStorage) Store(ctx context.Context, taskName string, scheduleAt time.Time, payload []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO task_scheduler
	(id,name,payload,scheduled_at)
	VALUES(?,?,?,?)`,
		uuid.New(), taskName, string(payload), scheduleAt.Format(time.DateTime))
	if err != nil {
		return err
	}
	return nil
}

func (s sqlStorage) Retrieve(ctx context.Context) (string, string, []byte, error) {
	var task taskModel
	err := s.db.GetContext(ctx, &task, "SELECT * FROM task_scheduler WHERE scheduled_at < '?' LIMIT 1", time.Now().Format(time.DateTime))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil, nil
		}
		return "", "", nil, err
	}
	return task.ID, task.Name, []byte(task.Payload), nil
}

func (s sqlStorage) Failure(ctx context.Context, taskID string) error {
	return nil
}

func (s sqlStorage) Done(ctx context.Context, taskID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM task_scheduler WHERE id=?", taskID)
	return err
}
