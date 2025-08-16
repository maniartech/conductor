package sequential

import (
	"context"
	stdErrors "errors"
	"testing"
	"time"

	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	internalErrors "github.com/maniartech/orchestrator/pkg/errors"
)

// CollectAll with mixed success/fail
func TestSequentialBuilder_Execute_CollectAll_Mixed(t *testing.T) {
	seq := Sequential(
		task.Task(func() (string, error) { return "A", nil }).Named("a"),
		task.Task(func() (int, error) { return 0, stdErrors.New("fail B") }).Named("b"),
		task.Task(func() (bool, error) { return true, nil }).Named("c"),
		task.Task(func() (string, error) { return "", stdErrors.New("fail D") }).Named("d"),
	)
	res, err := seq.Execute(context.Background(), config.Config{ErrorStrategy: internalErrors.CollectAll})
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	if len(res.Errors()) != 2 {
		t.Errorf("expected 2 errors got %d", len(res.Errors()))
	}
	if res.Get("a") != "A" || res.Get("c") != true {
		t.Error("successful results missing")
	}
}

// CollectAll with panic recovery in one step
func TestSequentialBuilder_Execute_CollectAll_Panic(t *testing.T) {
	seq := Sequential(
		task.Task(func() (string, error) { return "A", nil }).Named("a"),
		task.Task(func() (int, error) { panic("boom") }).Named("panic-step"),
		task.Task(func() (string, error) { return "C", nil }).Named("c"),
	)
	res, err := seq.Execute(context.Background(), config.Config{ErrorStrategy: internalErrors.CollectAll})
	if err == nil {
		t.Fatal("expected error due to panic")
	}
	if len(res.Errors()) != 1 {
		t.Errorf("expected 1 panic error got %d", len(res.Errors()))
	}
	if res.Get("a") != "A" || res.Get("c") != "C" {
		t.Error("expected other results present")
	}
}

// CollectAll with context cancellation mid-way
func TestSequentialBuilder_Execute_CollectAll_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	seq := Sequential(
		task.Task(func() (string, error) { time.Sleep(10 * time.Millisecond); cancel(); return "one", nil }).Named("one"),
		task.Task(func() (string, error) { time.Sleep(100 * time.Millisecond); return "two", nil }).Named("two"),
		task.Task(func() (string, error) { return "three", nil }).Named("three"),
	)
	_, err := seq.Execute(ctx, config.Config{ErrorStrategy: internalErrors.CollectAll})
	if err == nil {
		t.Error("expected cancellation error")
	}
}

// CollectAll with timeout (uses provided config timeout)
func TestSequentialBuilder_Execute_CollectAll_Timeout(t *testing.T) {
	seq := Sequential(
		task.Task(func() (string, error) { time.Sleep(50 * time.Millisecond); return "slow", nil }).Named("slow"),
		task.Task(func() (string, error) { return "fast", nil }).Named("fast"),
	)
	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: internalErrors.CollectAll, Timeout: 10 * time.Millisecond}
	_, err := seq.Execute(ctx, cfg)
	if err == nil {
		t.Error("expected timeout error")
	}
}
