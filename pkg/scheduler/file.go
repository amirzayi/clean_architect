package scheduler

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const taskDirectory = "task_scheduler"
const executingTaskDirectory = "executing_task_scheduler"
const failedTaskDirectory = "failed_task_scheduler"

type file struct{}

func NewFileScheduler() Storage {
	return file{}
}

func (file) writeFile(filePath string, payload []byte) error {
	if _, err := os.Stat(filepath.Dir(filePath)); os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
	}
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	_, err = f.Write(payload)
	return err
}

func (f file) Store(_ context.Context, taskName string, scheduleAt time.Time, payload []byte) error {
	filePath := filepath.Join(taskDirectory, fmt.Sprintf("%s_%s.task", taskName, scheduleAt.Format(time.RFC3339Nano)))
	return f.writeFile(filePath, payload)
}

func (f file) Retrieve(ctx context.Context) (string, string, []byte, error) {
	files, err := filepath.Glob(filepath.Join(taskDirectory, "*.task"))
	if err != nil {
		return "", "", nil, err
	}
	if len(files) < 1 {
		return "", "", nil, nil
	}

	slices.Sort(files)
	firstFilePath := files[0]
	fileName := filepath.Base(firstFilePath)
	taskName := strings.Split(fileName, "_")[0]
	firstFile, err := os.Open(firstFilePath)
	if err != nil {
		return "", "", nil, err
	}
	data, err := io.ReadAll(firstFile)
	if err != nil {
		return "", "", nil, err
	}
	executingFilePath := filepath.Join(executingTaskDirectory, fileName)
	if err = f.writeFile(executingFilePath, data); err != nil {
		return "", "", nil, err
	}
	if err = os.Remove(firstFilePath); err != nil {
		return "", "", nil, err
	}
	return fileName, taskName, data, nil
}

func (f file) Failure(ctx context.Context, fileName string) error {
	filePath := filepath.Join(executingTaskDirectory, fileName)
	executing, err := os.Open(filePath)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(executing)
	if err != nil {
		return err
	}
	err = f.writeFile(filepath.Join(failedTaskDirectory, fileName), data)
	if err != nil {
		return err
	}
	return os.Remove(filePath)
}

func (file) Done(ctx context.Context, fileName string) error {
	filePath := filepath.Join(executingTaskDirectory, fileName)
	return os.Remove(filePath)
}
