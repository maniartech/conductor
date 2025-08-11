package concurrent

import (
	"context"
	stdErrors "errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	internalErrors "github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/task"
)

func TestConcurrent_Execute_FailFast_Success(t *testing.T) {
	c := Concurrent(
		// ensure parallelism
		task.Task(func() (int, error) { time.Sleep(5 * time.Millisecond); return 1, nil }).Named("one"),
		task.Task(func() (int, error) { return 2, nil }).Named("two"),
	)
	res, err := c.Execute(context.Background(), config.Config{ErrorStrategy: internalErrors.FailFast})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if res.Get("one") != 1 || res.Get("two") != 2 {
		t.Error("unexpected results")
	}
}

func TestConcurrent_Execute_FailFast_FirstError(t *testing.T) {
	boom := stdErrors.New("boom")
	var secondStarted int32
	c := Concurrent(
		task.Task(func() (string, error) { return "", boom }),
		task.Task(func() (int, error) {
			atomic.AddInt32(&secondStarted, 1)
			time.Sleep(100 * time.Millisecond)
			return 42, nil
		}),
	).ErrorBoundary(internalErrors.FailFast)
	res, err := c.Execute(context.Background(), config.DefaultConfig())
	if err == nil {
		t.Fatal("expected error")
	}
	if len(res.Errors()) == 0 {
		t.Fatalf("expected at least one error")
	}
	if res.Errors()[0].Error.Error() != "boom" {
		t.Errorf("first error should be boom, got %v", res.Errors()[0].Error)
	}
}

func TestConcurrent_Execute_FailFast_Cancellation(t *testing.T) {
	var started int32
	ctx, cancel := context.WithCancel(context.Background())
	c := Concurrent(
		task.Task(func() (string, error) { atomic.AddInt32(&started, 1); cancel(); return "early", nil }),
		task.Task(func() (string, error) { time.Sleep(40 * time.Millisecond); return "late", nil }),
	).ErrorBoundary(internalErrors.FailFast)
	_, err := c.Execute(ctx, config.DefaultConfig())
	if err == nil {
		t.Error("expected cancellation error")
	}
	if atomic.LoadInt32(&started) == 0 {
		t.Error("first task not started")
	}
}
