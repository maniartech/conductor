package concurrent

import (
	"context"
	"testing"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/pkg/builders/task"

	. "github.com/maniartech/orchestrator/pkg/builders/concurrent"
)

func TestConcurrent_ChildAccess(t *testing.T) {
	c := Concurrent(
		// use task.Task directly; Named returns types.Orchestration
		task.Task(func() (int, error) { return 1, nil }).Named("one"),
		task.Task(func() (int, error) { return 2, nil }).Named("two"),
	)

	if c.GetChildCount() != 2 {
		t.Fatalf("expected 2 children got %d", c.GetChildCount())
	}

	// GetChildNames
	names := c.GetChildNames()
	if len(names) != 2 || names[0] != "one" || names[1] != "two" {
		t.Fatalf("unexpected names %#v", names)
	}

	// GetChildAt success
	ch0, err := c.GetChildAt(0)
	if err != nil || ch0 == nil || ch0.GetName() != "one" {
		t.Fatalf("unexpected first child: %v %v", ch0, err)
	}
	// GetChildAt out of bounds
	if _, err = c.GetChildAt(5); err == nil {
		t.Error("expected bounds error")
	}

	// FindChildByName existing
	idx, found := c.FindChildByName("two")
	if idx != 1 || found == nil || found.GetName() != "two" {
		t.Fatalf("could not find child 'two'")
	}
	// FindChildByName missing
	if idx, found = c.FindChildByName("missing"); idx != -1 || found != nil {
		t.Errorf("expected not found for missing name")
	}

	// GetChildren returns copy
	children := c.GetChildren()
	if len(children) != 2 {
		t.Fatalf("expected 2 children copy got %d", len(children))
	}
	children[0] = nil // should not affect original
	orig, _ := c.GetChildAt(0)
	if orig == nil {
		t.Error("external modification affected internal slice")
	}

	res, err := c.Execute(context.Background(), config.DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if res.Get("one") != 1 || res.Get("two") != 2 {
		t.Errorf("unexpected execution results: one=%v two=%v", res.Get("one"), res.Get("two"))
	}
}
