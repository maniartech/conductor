package orchestration

import (
	"context"
	"testing"

	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/result"
	"github.com/maniartech/orchestrator/pkg/types"

	. "github.com/maniartech/orchestrator/internal/orchestration"
)

type nestedMock struct {
	*BaseOrchestrationBuilder
	resolver *PathResolverBase
	children []types.Orchestration
}

func newNestedMock(tpe, name string, children ...types.Orchestration) *nestedMock {
	b := NewBaseOrchestrationBuilder(tpe)
	if name != "" {
		b.SetName(name)
	}
	nm := &nestedMock{BaseOrchestrationBuilder: b, children: children, resolver: NewPathResolverBase()}
	nm.resolver.SetCallbacks(
		func() string { return nm.GetCurrentPath() },
		func() []types.Orchestration { return nm.children },
		func(child types.Orchestration, idx int) string { return child.GetName() },
	)
	return nm
}

func (n *nestedMock) Named(name string) types.Orchestration    { n.SetName(name); return n }
func (n *nestedMock) With(c config.Config) types.Orchestration { n.SetConfig(c); return n }
func (n *nestedMock) ErrorBoundary(s errors.ErrorStrategy) types.Orchestration {
	n.SetErrorBoundary(s)
	return n
}
func (n *nestedMock) Execute(ctx context.Context, c config.Config) (*result.Result, error) {
	return result.NewResult(), nil
}

// Path methods (override to use resolver recursion where appropriate)
func (n *nestedMock) GetByPath(p string) (types.Orchestration, error) {
	if p == n.GetCurrentPath() {
		return n, nil
	}
	return n.resolver.GetByPath(p)
}
func (n *nestedMock) GetCurrentPath() string { return n.BaseOrchestrationBuilder.GetCurrentPath(n) }
func (n *nestedMock) ListAllPaths() []string { return n.resolver.ListAllPaths() }
func (n *nestedMock) FindByName(name string) []types.PathMatch {
	matches := n.resolver.FindByName(name)
	if n.GetName() == name {
		matches = append([]types.PathMatch{{Path: n.GetCurrentPath(), Orchestration: n, Depth: 0, Type: n.BaseOrchestrationBuilder.orchestrationType}}, matches...)
	}
	return matches
}
func (n *nestedMock) GetOrchestrationTree() *types.OrchestrationTree {
	tree := n.resolver.GetOrchestrationTree()
	if tree != nil {
		tree.Orchestration = n
	}
	return tree
}
func (n *nestedMock) Query() *types.PathQuery { return n.resolver.Query() }

// Build hierarchy: root(sequential) -> mid1(dup), mid2 -> leaf1(dup), leaf2
func buildNestedTree() *nestedMock {
	leaf1 := newNestedMock("task", "dup")
	leaf2 := newNestedMock("task", "leaf2")
	mid1 := newNestedMock("sequential", "dup", leaf1)
	mid2 := newNestedMock("sequential", "mid2", leaf2)
	root := newNestedMock("concurrent", "root", mid1, mid2)
	return root
}

func TestPathResolverBase_RecursiveTreeAndQuery(t *testing.T) {
	root := buildNestedTree()
	paths := root.ListAllPaths()
	if len(paths) < 5 {
		t.Fatalf("expected >=5 paths got %v", paths)
	}
	var foundDup bool
	for _, p := range paths {
		if contains(p, "dup") {
			foundDup = true
			break
		}
	}
	if !foundDup {
		t.Errorf("expected a dup path in %v", paths)
	}

	dupMatches := root.FindByName("dup")
	if len(dupMatches) < 2 {
		t.Fatalf("expected >=2 dup matches got %d", len(dupMatches))
	}

	q := root.Query()
	pat := q.FindByPattern("*dup*")
	if len(pat) < 2 {
		t.Errorf("expected pattern matches for dup got %d", len(pat))
	}

	leafs := q.FindLeafNodes()
	if len(leafs) == 0 {
		t.Error("expected leaf nodes")
	}
}

// Removed duplicate contains helper (already defined in behavior test)
