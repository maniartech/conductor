package sequential

import (
	"context"
	stdErrors "errors"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	internalErrors "github.com/maniartech/orchestrator/internal/errors"
	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/types"
)

// Success path with FailFast
func TestSequentialBuilder_Execute_FailFast_Success(t *testing.T) {
	seq := Sequential(
		task.Task(func() (string, error) { return "a", nil }).Named("first"),
		task.Task(func() (int, error) { return 2, nil }).Named("second"),
	)
	res, err := seq.Execute(context.Background(), config.Config{ErrorStrategy: internalErrors.FailFast})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Get("first") != "a" || res.Get("second") != 2 {
		t.Errorf("unexpected results: %v", res)
	}
	if seq.GetStatus() != types.Completed {
		t.Errorf("expected Completed got %v", seq.GetStatus())
	}
}

// First task fails -> stop immediately
func TestSequentialBuilder_Execute_FailFast_FirstError(t *testing.T) {
	boom := stdErrors.New("boom")
	seq := Sequential(
		task.Task(func() (string, error) { return "", boom }).Named("fail"),
		task.Task(func() (int, error) { t.Fatalf("second should not run"); return 0, nil }),
	)
	res, err := seq.Execute(context.Background(), config.Config{ErrorStrategy: internalErrors.FailFast})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(res.Errors()) != 1 {
		t.Errorf("expected 1 error got %d", len(res.Errors()))
	}
	if res.Get("fail") != nil {
		t.Error("should not set result for failing step")
	}
}

// Cancellation before second step
func TestSequentialBuilder_Execute_FailFast_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	seq := Sequential(
		task.Task(func() (string, error) { cancel(); return "done", nil }),
		task.Task(func() (int, error) { time.Sleep(50 * time.Millisecond); return 1, nil }),
	)
	_, err := seq.Execute(ctx, config.Config{ErrorStrategy: internalErrors.FailFast})
	if err == nil {
		t.Error("expected cancellation error")
	}
}
