package task

import (
	"context"
	systemErrors "errors"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	. "github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/types"
)

func TestTaskBuilderExecute(t *testing.T) {
	tests := []struct {
		name        string
		fn          func() (interface{}, error)
		expected    interface{}
		expectError bool
		nameAssign  string
	}{
		{"successful execution", func() (interface{}, error) { return "success", nil }, "success", false, "success-task"},
		{"execution with error", func() (interface{}, error) { return nil, systemErrors.New("task error") }, nil, true, "error-task"},
		{"execution with panic", func() (interface{}, error) { panic("task panic") }, nil, true, "panic-task"},
	}
	for _, tc := range tests {
		c := tc
		t.Run(c.name, func(t *testing.T) {
			tsk := Task(c.fn).Named(c.nameAssign)
			res, err := tsk.Execute(context.Background(), config.DefaultConfig())
			if c.expectError {
				if err == nil {
					t.Error("expected error")
				}
				if res == nil || !res.HasErrors() {
					t.Fatal("expected result with errors")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				if res == nil || res.HasErrors() {
					t.Fatal("unexpected errors in result")
				}
				if c.expected != nil && res.Get(c.nameAssign) != c.expected {
					t.Errorf("expected %v got %v", c.expected, res.Get(c.nameAssign))
				}
			}
		})
	}
}

func TestTaskBuilderExecuteWithTimeout(t *testing.T) {
	tsk := Task(func() (string, error) { time.Sleep(100 * time.Millisecond); return "completed", nil }).Named("timeout-task")
	res, err := tsk.Execute(context.Background(), config.Config{Timeout: 50 * time.Millisecond})
	if err == nil {
		t.Error("expected timeout error")
	}
	if !systemErrors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded got %v", err)
	}
	if res == nil || !res.HasErrors() {
		t.Fatal("expected result with errors on timeout")
	}
}

func TestTaskBuilderExecuteWithCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tsk := Task(func() (string, error) { return "completed", nil }).Named("cancel-task")
	res, err := tsk.Execute(ctx, config.DefaultConfig())
	if err == nil {
		t.Error("expected cancellation error")
	}
	if !systemErrors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled got %v", err)
	}
	if res == nil || !res.HasErrors() {
		t.Fatal("expected result with errors")
	}
}

func TestTaskBuilderPanicRecoveryWithStackTrace(t *testing.T) {
	tsk := Task(func() (string, error) { panic("test panic for stack trace") }).Named("panic-task")
	res, err := tsk.Execute(context.Background(), config.DefaultConfig())
	if err == nil {
		t.Error("expected error from panic recovery")
	}
	if !contains(err.Error(), "test panic for stack trace") {
		t.Errorf("missing panic message in error: %v", err)
	}
	if !contains(err.Error(), "Stack trace:") {
		t.Errorf("missing stack trace in error: %v", err)
	}
	if res == nil || !res.HasErrors() {
		t.Fatal("expected errors in result")
	}
	opErrs := res.Errors()
	if len(opErrs) != 1 {
		t.Fatalf("expected 1 error got %d", len(opErrs))
	}
	if len(opErrs[0].Stack) == 0 {
		t.Error("expected stack bytes")
	}
}

func TestTaskBuilderTimeoutHandling(t *testing.T) {
	tsk := Task(func() (string, error) { time.Sleep(100 * time.Millisecond); return "completed", nil }).Named("timeout-task")
	start := time.Now()
	res, err := tsk.Execute(context.Background(), config.Config{Timeout: 50 * time.Millisecond})
	dur := time.Since(start)
	if err == nil {
		t.Error("expected timeout")
	}
	if !systemErrors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded got %v", err)
	}
	if dur > 80*time.Millisecond {
		t.Errorf("timeout took too long: %v", dur)
	}
	if tsk.GetStatus() != types.Cancelled {
		t.Errorf("expected Cancelled got %v", tsk.GetStatus())
	}
	if res == nil || len(res.Errors()) != 1 {
		t.Fatalf("expected single error result")
	}
	if res.Errors()[0].Duration <= 0 {
		t.Error("expected positive duration metadata")
	}
}

func TestTaskBuilderCancellationHandling(t *testing.T) {
	tsk := Task(func() (string, error) { return "completed", nil }).Named("cancel-task")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := tsk.Execute(ctx, config.DefaultConfig())
	if err == nil {
		t.Error("expected cancellation error")
	}
	if !systemErrors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled got %v", err)
	}
	if tsk.GetStatus() != types.Cancelled {
		t.Errorf("expected Cancelled got %v", tsk.GetStatus())
	}
	if res == nil || !res.HasErrors() {
		t.Fatal("expected errors in result")
	}
}
