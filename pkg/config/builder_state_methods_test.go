package config

import (
	"context"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/errors"
)

// These tests specifically target small helper/state methods that were previously
// untested: HasErrors(), Errors(), and additional inheritance context branches.

func TestConfigBuilderHasErrorsAndErrorsSlice(t *testing.T) {
	b := NewConfigBuilder()

	if b.HasErrors() {
		t.Fatalf("expected no errors initially")
	}
	if errs := b.Errors(); errs != nil { // should return nil when no errors recorded
		t.Fatalf("expected nil slice when there are no errors, got %v", errs)
	}

	// Introduce multiple validation errors via fluent API (these accumulate in builder.errors directly)
	b.Timeout(0).MaxConcurrency(0).Retries(-1) // three distinct errors

	if !b.HasErrors() {
		t.Fatalf("expected HasErrors() to be true after introducing errors")
	}

	errs := b.Errors()
	if len(errs) != 3 { // ensure we captured all three
		t.Fatalf("expected 3 errors, got %d", len(errs))
	}

	// Ensure returned slice is a copy (mutating it should not affect internal state)
	errs[0] = nil
	again := b.Errors()
	if len(again) != 3 || again[0] == nil { // internal slice should remain intact
		t.Fatalf("expected internal errors slice to be unchanged after external mutation attempt")
	}

	// Reset should clear errors
	b.Reset()
	if b.HasErrors() || b.Errors() != nil {
		t.Fatalf("expected errors cleared after Reset()")
	}
}

func TestConfigInheritContextOverrideAndPreserve(t *testing.T) {
	parentCtx, parentCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer parentCancel()
	childCtx, childCancel := context.WithCancel(context.Background())
	defer childCancel()

	parent := Config{ // parent context set
		ErrorStrategy:  errors.CollectAll,
		Timeout:        45 * time.Second,
		Retries:        2,
		MaxConcurrency: 500,
		Context:        parentCtx,
	}

	// Case 1: Child does not specify context -> inherits parent's context
	inherit1 := (Config{Timeout: 5 * time.Second}).Inherit(parent)
	if inherit1.Context != parentCtx {
		t.Fatalf("expected parent context to be inherited when child context is nil")
	}
	if inherit1.Timeout != 5*time.Second { // ensure other overrides still applied
		t.Fatalf("expected child timeout override applied, got %v", inherit1.Timeout)
	}

	// Case 2: Child provides its own context which should override parent
	inherit2 := (Config{Context: childCtx}).Inherit(parent)
	if inherit2.Context != childCtx {
		t.Fatalf("expected child context to override parent context")
	}
	if inherit2.ErrorStrategy != errors.CollectAll { // ensure inherited fields preserved
		t.Fatalf("expected ErrorStrategy inherited from parent, got %v", inherit2.ErrorStrategy)
	}
}
