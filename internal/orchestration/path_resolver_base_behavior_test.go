package orchestration

import (
	"errors"
	"testing"

	"github.com/maniartech/orchestrator/pkg/types"
)

type simpleChild struct{ types.Orchestration }

// createMockTree builds: root (leaf-a, child-b(inner1))
func createMockTree() *PathResolverBase {
	leafA := NewMockOrchestration("task").Named("leaf-a")
	leafInner := NewMockOrchestration("task").Named("inner1")

	childB := NewMockOrchestration("sequential").Named("child-b")
	childBResolver := NewPathResolverBase()
	childBResolver.SetCallbacks(
		childB.GetCurrentPath,
		func() []types.Orchestration { return []types.Orchestration{leafInner} },
		func(ch types.Orchestration, idx int) string { return ch.GetName() },
	)

	root := NewMockOrchestration("concurrent").Named("root")
	rootResolver := NewPathResolverBase()
	children := []types.Orchestration{leafA, childB}
	rootResolver.SetCallbacks(
		root.GetCurrentPath,
		func() []types.Orchestration { return children },
		func(ch types.Orchestration, idx int) string { return ch.GetName() },
	)
	return rootResolver
}

func TestPathResolverBase_ListAllPaths_And_FindByName(t *testing.T) {
	resolver := createMockTree()
	paths := resolver.ListAllPaths()
	if len(paths) < 3 {
		t.Fatalf("expected at least 3 paths got %v", paths)
	}
	if len(resolver.FindByName("leaf-a")) == 0 {
		t.Fatalf("expected match for leaf-a")
	}
	if len(resolver.FindByName("child-b")) == 0 {
		t.Fatalf("expected match for child-b")
	}
}

func TestPathResolverBase_GetByPath_SuccessAndErrors(t *testing.T) {
	resolver := createMockTree()
	paths := resolver.ListAllPaths()
	var leafPath string
	for _, p := range paths {
		if contains(p, "leaf-a") {
			leafPath = p
			break
		}
	}
	if leafPath == "" {
		t.Fatal("leaf-a path not found in list")
	}
	if _, err := resolver.GetByPath(leafPath); err != nil {
		t.Errorf("expected success for direct child path got %v", err)
	}
	// deeper path beyond leaf should error
	if _, err := resolver.GetByPath(leafPath + ".extra"); err == nil {
		t.Error("expected error for deeper segment beyond leaf")
	}
	if _, err := resolver.GetByPath("unknown.path"); err == nil {
		t.Error("expected prefix error")
	}
}

func TestPathResolverBase_Uninitialized(t *testing.T) {
	prb := &PathResolverBase{}
	if _, err := prb.GetByPath("anything"); err == nil {
		t.Error("expected init error")
	}
	if res := prb.ListAllPaths(); len(res) != 0 {
		t.Errorf("expected empty paths for uninitialized got %v", res)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

var _ = errors.New
