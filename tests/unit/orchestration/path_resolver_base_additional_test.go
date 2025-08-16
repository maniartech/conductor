package orchestration

import (
	"testing"

	. "github.com/maniartech/orchestrator/internal/orchestration"
)

// TestPathResolverBase_Callbacks_Minimal ensures SetCallbacks with the new 3-arg signature works.
func TestPathResolverBase_Callbacks_Minimal(t *testing.T) {
	prb := NewPathResolverBase()
	called := false
	prb.SetCallbacks(
		func() string { return "root" },
		func() []Orchestration { return []Orchestration{} },
		func(child Orchestration, index int) string { called = true; return "child" },
	)

	if prb.GetCurrentPath() != "root" {
		t.Fatalf("expected current path root")
	}

	paths := prb.ListAllPaths()
	if len(paths) != 1 || paths[0] != "root" {
		t.Fatalf("expected only root path, got %v", paths)
	}

	// trigger getChildName indirectly (no children so it won't be called)
	_ = called // just assert variable exists
}
