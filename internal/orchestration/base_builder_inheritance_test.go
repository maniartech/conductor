package orchestration

import (
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/errors"
)

func TestBaseOrchestrationBuilder_ApplyConfigurationInheritance_LocalOverrideAndParentIntact(t *testing.T) {
	parent := config.Config{Timeout: 5 * time.Second, ErrorStrategy: errors.FailFast, MaxConcurrency: 3}
	child := NewBaseOrchestrationBuilder("sequential")
	child.SetConfig(config.Config{Timeout: 10 * time.Second})
	final := child.ApplyConfigurationInheritance(parent)
	if final.Timeout != 10*time.Second {
		t.Errorf("expected override timeout 10s got %v", final.Timeout)
	}
	if final.MaxConcurrency != 3 {
		t.Errorf("expected inherited max concurrency 3 got %d", final.MaxConcurrency)
	}
	if parent.Timeout != 5*time.Second {
		t.Errorf("parent mutated timeout %v", parent.Timeout)
	}
}

func TestBaseOrchestrationBuilder_ApplyConfigurationInheritance_ErrorBoundaryOverrideExisting(t *testing.T) {
	parent := config.Config{ErrorStrategy: errors.FailFast}
	child := NewBaseOrchestrationBuilder("task")
	child.SetConfig(config.Config{Timeout: 1 * time.Second, ErrorStrategy: errors.FailFast})
	child.SetErrorBoundary(errors.CollectAll)
	final := child.ApplyConfigurationInheritance(parent)
	if final.ErrorStrategy != errors.CollectAll {
		t.Errorf("expected boundary override CollectAll got %v", final.ErrorStrategy)
	}
}

func TestBaseOrchestrationBuilder_ApplyConfigurationInheritance_NoLocalConfig(t *testing.T) {
	parent := config.Config{Timeout: 2 * time.Second, ErrorStrategy: errors.CollectAll, MaxConcurrency: 9}
	child := NewBaseOrchestrationBuilder("concurrent")
	final := child.ApplyConfigurationInheritance(parent)
	if final.Timeout != 2*time.Second || final.MaxConcurrency != 9 || final.ErrorStrategy != errors.CollectAll {
		t.Errorf("inheritance failed: %+v", final)
	}
}
