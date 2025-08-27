package concurrent

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/orchestrator"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	internalErrors "github.com/maniartech/orchestrator/pkg/errors"

	. "github.com/maniartech/orchestrator/pkg/builders/concurrent"
)

func TestConcurrent_Timeout_ConfigInheritance(t *testing.T) {
	c := Concurrent(
		task.Task(func(ctx orchestrator.Context) (string, error) { time.Sleep(30 * time.Millisecond); return "slow", nil }),
		task.Task(func(ctx orchestrator.Context) (string, error) { return "fast", nil }),
	).With(config.Config{Timeout: 15 * time.Millisecond})
	_, err := c.Execute(context.Background(), config.DefaultConfig())
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestConcurrent_Cancellation_Context(t *testing.T) {
	var started int32
	ctx, cancel := context.WithCancel(context.Background())
	c := Concurrent(
		task.Task(func(ctx orchestrator.Context) (string, error) {
			atomic.AddInt32(&started, 1)
			time.Sleep(20 * time.Millisecond)
			cancel()
			return "one", nil
		}),
		task.Task(func(ctx orchestrator.Context) (string, error) { time.Sleep(50 * time.Millisecond); return "two", nil }),
	).ErrorBoundary(internalErrors.FailFast)
	_, err := c.Execute(ctx, config.DefaultConfig())
	if err == nil {
		t.Error("expected cancellation error")
	}
	if atomic.LoadInt32(&started) == 0 {
		t.Error("first task not started")
	}
}
