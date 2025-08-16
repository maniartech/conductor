package task

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
)

func TestTaskBuilderErrorMetadata(t *testing.T) {
	tk := Task(func() (string, error) { time.Sleep(1 * time.Millisecond); return "", errors.New("test error") }).Named("metadata-task")
	cfg := config.DefaultConfig()
	start := time.Now()
	res, err := tk.Execute(context.Background(), cfg)
	end := time.Now()
	if err == nil {
		t.Fatal("expected error")
	}
	if res == nil {
		t.Fatal("nil result")
	}
	opErrs := res.Errors()
	if len(opErrs) != 1 {
		t.Fatalf("expected 1 error got %d", len(opErrs))
	}
	op := opErrs[0]
	if op.Error.Error() != "test error" {
		t.Errorf("unexpected error msg %v", op.Error)
	}
	if op.Index != 0 {
		t.Errorf("expected index 0 got %d", op.Index)
	}
	if op.Duration <= 0 {
		t.Error("expected positive duration")
	}
	if op.Timestamp.Before(start) || op.Timestamp.After(end) {
		t.Error("timestamp out of bounds")
	}
	if op.OpID != "task-metadata-task" {
		t.Errorf("unexpected OpID %s", op.OpID)
	}
	if len(op.Stack) == 0 {
		t.Error("expected stack")
	}
}
