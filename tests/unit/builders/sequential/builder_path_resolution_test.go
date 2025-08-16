package sequential

import (
	"context"
	"testing"

	internalErrors "github.com/maniartech/orchestrator/internal/errors"
	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
)

func TestSequentialBuilder_PathResolution(t *testing.T) {
	inner := Sequential(
		task.Task(func() (string, error) { return "x", nil }).Named("inner-task"),
	).Named("inner")
	outer := Sequential(
		task.Task(func() (string, error) { return "a", nil }).Named("alpha"),
		inner,
		task.Task(func() (string, error) { return "b", nil }),
	).Named("outer")

	// Execute to register results for tree resolution validations
	_, err := outer.Execute(context.Background(), config.Config{ErrorStrategy: internalErrors.FailFast})
	if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}

	// Current path non-empty
	if outer.GetCurrentPath() == "" {
		t.Error("expected non-empty current path")
	}

	// List all paths includes child
	paths := outer.ListAllPaths()
	found := false
	for _, p := range paths {
		if len(p) > 0 && p == outer.GetCurrentPath()+".inner" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find inner path")
	}

	// FindByName includes both outer and inner-task
	matches := outer.FindByName("inner-task")
	if len(matches) == 0 {
		t.Error("expected match for inner-task")
	}

	// Tree shape
	tree := outer.GetOrchestrationTree()
	if tree == nil || len(tree.Children) != 3 {
		t.Fatalf("unexpected tree structure: %#v", tree)
	}
}
