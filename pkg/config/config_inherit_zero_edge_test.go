package config

import (
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/errors"
)

// Focus: exercise subtle zero-value inheritance branches where parent itself has zero values.
// Although DefaultConfig never produces these, defensively validate logic when parent fields are zero.
func TestConfigInheritWhenParentZeroValues(t *testing.T) {
	// parent with zero (invalid) values to ensure child's non-zero overrides still apply correctly
	parent := Config{} // all zero (ErrorStrategy=0 -> FailFast by constant definition but treat as zero path in logic)

	// Child overrides each field one-by-one to ensure override conditions: (x != 0 || parent.x == 0)
	child := Config{
		ErrorStrategy:  errors.CollectAll,
		Timeout:        2 * time.Second,
		Retries:        4,
		MaxConcurrency: 250,
		// Context left nil to verify it stays nil because both parent & child are nil
	}

	res := child.Inherit(parent)
	if res.ErrorStrategy != errors.CollectAll {
		t.Fatalf("expected child ErrorStrategy override; got %v", res.ErrorStrategy)
	}
	if res.Timeout != 2*time.Second {
		t.Fatalf("expected child Timeout override; got %v", res.Timeout)
	}
	if res.Retries != 4 {
		t.Fatalf("expected child Retries override; got %d", res.Retries)
	}
	if res.MaxConcurrency != 250 {
		t.Fatalf("expected child MaxConcurrency override; got %d", res.MaxConcurrency)
	}
	if res.Context != nil { // child context nil should stay nil because child did not set, parent also nil
		t.Fatalf("expected resulting Context to remain nil when both parent & child contexts are nil")
	}
}

// Additional branch: child provides only partial overrides with parent explicit zeros
func TestConfigPartialOverrideWithZeroParent(t *testing.T) {
	parent := Config{} // zeros
	child := Config{Timeout: 10 * time.Second}

	res := child.Inherit(parent)
	if res.Timeout != 10*time.Second {
		t.Fatalf("expected timeout override; got %v", res.Timeout)
	}
	// other fields should remain zero
	if res.Retries != 0 || res.MaxConcurrency != 0 || res.ErrorStrategy != 0 || res.Context != nil {
		t.Fatalf("unexpected non-zero inherited fields: %+v", res)
	}
}
