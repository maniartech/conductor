package task

import (
	"testing"

	. "github.com/maniartech/orchestrator/internal/task"
)

func TestTask_PathResolutionCoverage(t *testing.T) {
	tt := Task(func() (string, error) { return "x", nil }).Named("path-task")
	path := tt.GetCurrentPath()
	if path == "" {
		t.Fatalf("expected non-empty current path")
	}
	got, err := tt.GetByPath(path)
	if err != nil || got == nil {
		t.Fatalf("expected retrieve by path err=%v", err)
	}
	if _, err = tt.GetByPath(path + "-missing"); err == nil {
		t.Fatalf("expected error for missing path")
	}
	paths := tt.ListAllPaths()
	if len(paths) != 1 || paths[0] != path {
		t.Fatalf("expected single path %s got %v", path, paths)
	}
	matches := tt.FindByName("path-task")
	if len(matches) != 1 || matches[0].Path != path {
		t.Fatalf("expected match for name path-task got %v", matches)
	}
	tree := tt.GetOrchestrationTree()
	if tree == nil || tree.Path != path || len(tree.Children) != 0 {
		t.Fatalf("unexpected tree %+v", tree)
	}
	q := tt.Query()
	if q == nil {
		t.Fatalf("expected non-nil query")
	}
	leaf := q.FindLeafNodes()
	if len(leaf) == 0 {
		t.Fatalf("expected leaf nodes")
	}
}
