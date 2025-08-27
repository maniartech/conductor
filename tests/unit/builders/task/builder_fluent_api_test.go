package task

import (
	"testing"
	"time"

	"github.com/maniartech/orchestrator"
	. "github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/errors"
)

func TestTaskBuilderNamed(t *testing.T) {
	task := Task(func(ctx orchestrator.Context) (string, error) { return "test", nil })
	res := task.Named("test-task")
	if res.(*TaskBuilder[string]) != task {
		t.Error("Named should return same instance")
	}
	if task.GetName() != "test-task" {
		t.Errorf("expected name test-task got %s", task.GetName())
	}
}

func TestTaskBuilderWith(t *testing.T) {
	task := Task(func(ctx orchestrator.Context) (string, error) { return "test", nil })
	cfg := config.Config{ErrorStrategy: errors.CollectAll, Timeout: 30 * time.Second, MaxConcurrency: 50}
	res := task.With(cfg)
	if res.(*TaskBuilder[string]) != task {
		t.Error("With should return same instance")
	}
	if task.GetConfig() == nil || task.GetConfig().ErrorStrategy != errors.CollectAll {
		t.Error("config not applied")
	}
}

func TestTaskBuilderErrorBoundary(t *testing.T) {
	task := Task(func(ctx orchestrator.Context) (string, error) { return "test", nil })
	res := task.ErrorBoundary(errors.CollectAll)
	if res.(*TaskBuilder[string]) != task {
		t.Error("ErrorBoundary same instance")
	}
	if task.GetConfig() == nil || task.GetConfig().ErrorStrategy != errors.CollectAll {
		t.Error("boundary not set")
	}
	task.With(config.Config{Timeout: time.Second}).ErrorBoundary(errors.FailFast)
	if task.GetConfig().ErrorStrategy != errors.FailFast {
		t.Error("boundary not updated")
	}
}

func TestTaskBuilderFluentAPI(t *testing.T) {
	res := Task(func(ctx orchestrator.Context) (string, error) { return "x", nil }).Named("chain").With(config.Config{Timeout: time.Second}).ErrorBoundary(errors.CollectAll)
	task := res.(*TaskBuilder[string])
	if task.GetName() != "chain" || task.GetConfig() == nil || task.GetConfig().Timeout <= 0 || task.GetConfig().ErrorStrategy != errors.CollectAll {
		t.Error("fluent chain failed")
	}
}
