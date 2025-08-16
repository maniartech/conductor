package concurrent

import (
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	internalErrors "github.com/maniartech/orchestrator/pkg/errors"

	. "github.com/maniartech/orchestrator/pkg/builders/concurrent"
)

func TestConcurrentBuilder_FluentAPI(t *testing.T) {
	c := Concurrent(task.Task(func() (string, error) { return "a", nil }), task.Task(func() (int, error) { return 1, nil }))

	returned := c.Named("pipeline")
	if returned != c {
		t.Error("Named should return same instance")
	}
	if c.GetName() != "pipeline" {
		t.Errorf("expected name pipeline got %s", c.GetName())
	}

	cfg := config.Config{Timeout: 5 * time.Second, MaxConcurrency: 3, ErrorStrategy: internalErrors.FailFast}
	returned = c.With(cfg)
	if returned != c || c.GetConfig() == nil || c.GetConfig().Timeout != 5*time.Second {
		t.Error("With not applied")
	}

	returned = c.ErrorBoundary(internalErrors.CollectAll)
	if returned != c {
		t.Error("ErrorBoundary must return same instance")
	}
	if c.GetConfig() == nil {
		t.Error("expected config still present")
	}
}
