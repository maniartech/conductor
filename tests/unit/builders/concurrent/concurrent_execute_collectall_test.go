package concurrent

import (
	"context"
	stdErrors "errors"
	"testing"
	"time"

	"github.com/maniartech/orchestrator"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	internalErrors "github.com/maniartech/orchestrator/pkg/errors"

	. "github.com/maniartech/orchestrator/pkg/builders/concurrent"
)

func TestConcurrent_Execute_CollectAll_Mixed(t *testing.T) {
	c := Concurrent(
		task.Task(func(ctx orchestrator.Context) (string, error) { return "A", nil }).Named("a"),
		task.Task(func(ctx orchestrator.Context) (int, error) { return 0, stdErrors.New("fail b") }).Named("b"),
		task.Task(func(ctx orchestrator.Context) (bool, error) { return true, nil }).Named("c"),
		task.Task(func(ctx orchestrator.Context) (string, error) { return "", stdErrors.New("fail d") }).Named("d"),
	).ErrorBoundary(internalErrors.CollectAll)
	res, err := c.Execute(context.Background(), config.DefaultConfig())
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	if len(res.Errors()) != 2 {
		t.Errorf("expected 2 errors got %d", len(res.Errors()))
	}
	if res.Get("a") != "A" || res.Get("c") != true {
		t.Error("expected successful results retained")
	}
}

func TestConcurrent_Execute_CollectAll_Panic(t *testing.T) {
	c := Concurrent(
		task.Task(func(ctx orchestrator.Context) (string, error) { return "A", nil }).Named("a"),
		task.Task(func(ctx orchestrator.Context) (int, error) { panic("boom") }),
		task.Task(func(ctx orchestrator.Context) (string, error) { return "C", nil }).Named("c"),
	).ErrorBoundary(internalErrors.CollectAll)
	res, err := c.Execute(context.Background(), config.DefaultConfig())
	if err == nil {
		t.Fatal("expected panic aggregated error")
	}
	if len(res.Errors()) != 1 {
		t.Errorf("expected 1 error got %d", len(res.Errors()))
	}
	if res.Get("a") != "A" || res.Get("c") != "C" {
		t.Error("other results missing")
	}
}

func TestConcurrent_Execute_CollectAll_Timeout(t *testing.T) {
	c := Concurrent(
		task.Task(func(ctx orchestrator.Context) (string, error) { time.Sleep(40 * time.Millisecond); return "slow", nil }),
		task.Task(func(ctx orchestrator.Context) (string, error) { return "fast", nil }),
	).ErrorBoundary(internalErrors.CollectAll)
	_, err := c.Execute(context.Background(), config.Config{ErrorStrategy: internalErrors.CollectAll, Timeout: 10 * time.Millisecond})
	if err == nil {
		t.Error("expected timeout error")
	}
}
