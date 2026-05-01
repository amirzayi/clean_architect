package scheduler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const taskDirectory = "task_scheduler"
const executingTaskDirectory = "executing_task_scheduler"
const failedTaskDirectory = "failed_task_scheduler"
const taskFileTimeFormat = time.RFC3339Nano

type file struct {
	mu *sync.Mutex
}

func NewFileScheduler() Storage {
	return file{mu: new(sync.Mutex)}
}

// checkDirExist will check directory exists to prevent write on not existed directory
func checkDirExist(filePath string) error {
	if _, err := os.Stat(filepath.Dir(filePath)); os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
	}
	return nil
}

func (file) Store(_ context.Context, taskName string, scheduleAt time.Time, payload []byte) error {
	filePath := filepath.Join(taskDirectory, fmt.Sprintf("%s_%s.task", taskName, scheduleAt.Format(taskFileTimeFormat)))
	if err := checkDirExist(filePath); err != nil {
		return err
	}
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	_, err = f.Write(payload)
	return err
}

func (f file) Retrieve(ctx context.Context) (string, string, []byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// search task file
	files, err := filepath.Glob(filepath.Join(taskDirectory, "*.task"))
	if err != nil {
		return "", "", nil, err
	}
	if len(files) < 1 {
		return "", "", nil, nil
	}

	slices.Sort(files)
	firstFilePath := files[0]

	// file name be like DoSomeJob_2026-05-01T19:20:39.35869046+03:30.task
	fileName := filepath.Base(firstFilePath)
	// task name is equal to DoSomeJob
	taskName := strings.Split(fileName, "_")[0]
	// try to parse schedule time of file name
	scheduleTime := strings.TrimRight(strings.TrimLeft(fileName, fmt.Sprintf("%s_", taskName)), ".task")
	scheduledAt, err := time.Parse(taskFileTimeFormat, scheduleTime)
	if err != nil {
		// schedule time is not standard, so ignore task and delete them to prevent proccess it later
		return "", "", nil, errors.Join(err, os.Remove(firstFilePath))
	}
	// prevent to execute task early
	if scheduledAt.After(time.Now()) {
		return "", "", nil, nil
	}
	executingFilePath := filepath.Join(executingTaskDirectory, fileName)
	if err = checkDirExist(executingFilePath); err != nil {
		return "", "", nil, err
	}
	// move task to temporary directory to prevent concurrent execution in another goroutines
	if err = os.Rename(firstFilePath, executingFilePath); err != nil {
		return "", "", nil, err
	}
	firstFile, err := os.Open(executingFilePath)
	if err != nil {
		return "", "", nil, err
	}
	data, err := io.ReadAll(firstFile)
	if err != nil {
		return "", "", nil, err
	}
	return fileName, taskName, data, nil
}

func (file) Failure(ctx context.Context, fileName string) error {
	filePath := filepath.Join(executingTaskDirectory, fileName)
	failedTaskFilePath := filepath.Join(failedTaskDirectory, fileName)
	if err := checkDirExist(failedTaskFilePath); err != nil {
		return err
	}
	return os.Rename(filePath, failedTaskFilePath)
}

func (file) Done(ctx context.Context, fileName string) error {
	filePath := filepath.Join(executingTaskDirectory, fileName)
	return os.Remove(filePath)
}
