package sequential

import (
	"testing"

	"github.com/maniartech/orchestrator"
	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/builders/task"
)

func TestSequentialBuilder_ChildrenAccess(t *testing.T) {
	seq := Sequential(
		task.Task(func(ctx orchestrator.Context) (string, error) { return "a", nil }).Named("first"),
		task.Task(func(ctx orchestrator.Context) (int, error) { return 2, nil }),
		task.Task(func(ctx orchestrator.Context) (bool, error) { return true, nil }).Named("third"),
	)
	builder := seq // already *SequentialBuilder

	if builder.GetChildCount() != 3 {
		t.Fatalf("expected 3 children, got %d", builder.GetChildCount())
	}

	child, err := builder.GetChildAt(0)
	if err != nil || child.GetName() != "first" {
		t.Errorf("unexpected child0: %v %v", child.GetName(), err)
	}
	if _, err = builder.GetChildAt(5); err == nil {
		t.Error("expected bounds error")
	}

	// GetChildren returns copy
	children := builder.GetChildren()
	if len(children) != 3 {
		t.Errorf("expected 3 children copy, got %d", len(children))
	}
	// modify slice should not affect original
	if len(children) > 0 {
		children[0] = nil
	}
	origFirst, _ := builder.GetChildAt(0)
	if origFirst == nil {
		t.Error("external modification affected internal slice")
	}

	names := builder.GetChildNames()
	expected := []string{"first", "step-1", "third"}
	if len(names) != len(expected) {
		t.Fatalf("unexpected name length %v", names)
	}
	for i, v := range expected {
		if names[i] != v {
			t.Errorf("expected name %s at %d got %s", v, i, names[i])
		}
	}

	idx, c := builder.FindChildByName("third")
	if idx != 2 || c == nil {
		t.Error("FindChildByName failed for named child")
	}
	idx, c = builder.FindChildByName("step-1")
	if idx != 1 || c == nil {
		t.Error("FindChildByName failed for generated name")
	}
	idx, c = builder.FindChildByName("missing")
	if idx != -1 || c != nil {
		t.Error("expected not found for missing name")
	}
}
