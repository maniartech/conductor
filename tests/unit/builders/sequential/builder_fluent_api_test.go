package sequential

import (
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/errors"
	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/builders/task"
)

func TestSequentialBuilder_FluentAPI(t *testing.T) {
	seq := Sequential(
		task.Task(func() (string, error) { return "x", nil }),
		task.Task(func() (int, error) { return 1, nil }),
	)

	// Named
	returned := seq.Named("pipeline")
	if returned != seq {
		t.Error("Named should return same instance")
	}
	if seq.GetName() != "pipeline" {
		t.Errorf("expected name pipeline, got %s", seq.GetName())
	}

	// With
	cfg := config.Config{Timeout: 10 * time.Second, MaxConcurrency: 5, ErrorStrategy: errors.FailFast}
	returned = seq.With(cfg)
	if returned != seq {
		t.Error("With should return same instance")
	}
	if seq.GetConfig() == nil || seq.GetConfig().Timeout != 10*time.Second {
		t.Errorf("config not applied: %+v", seq.GetConfig())
	}

	// ErrorBoundary
	returned = seq.ErrorBoundary(errors.CollectAll)
	if returned != seq {
		t.Error("ErrorBoundary should return same instance")
	}
	// TODO: Check Error Boundry
	// if seq.errorBoundary == nil || *seq.errorBoundary != errors.CollectAll {
	// 	t.Error("error boundary not set")
	// }

	// Chain
	seq2 := Sequential(task.Task(func() (string, error) { return "y", nil })).
		Named("chain").
		With(config.Config{Timeout: 5 * time.Second}).
		ErrorBoundary(errors.FailFast)
	if seq2.GetName() != "chain" {
		t.Error("chain name not applied")
	}
	if seq2.GetConfig() == nil || seq2.GetConfig().Timeout != 5*time.Second {
		t.Error("chain config not applied")
	}
}
