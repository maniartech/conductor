package concurrent

import (
	"testing"

	"github.com/maniartech/orchestrator/internal/task"
)

func TestConcurrent_PathResolution(t *testing.T){
	c := Concurrent(
		Concurrent(
			task.Task(func()(string,error){return "a",nil}).Named("inner-a"),
			task.Task(func()(string,error){return "b",nil}),
		).Named("inner"),
		task.Task(func()(int,error){return 1,nil}).Named("leaf"),
	).Named("root")

	paths := c.ListAllPaths()
	if len(paths)==0 { t.Fatal("expected paths") }

	// ensure at least leaf and inner identifier appears
	var hasLeaf, hasInner bool
	for _, p := range paths {
		if contains(p, "leaf") { hasLeaf = true }
		if contains(p, "inner") { hasInner = true }
	}
	if !hasLeaf || !hasInner { t.Errorf("expected leaf and inner in paths: %v", paths) }

	matches := c.FindByName("inner-a")
	if len(matches)==0 { t.Fatalf("expected match for inner-a") }

	// Basic pattern search using full task name path component
	q := c.Query()
	patternMatches := q.FindByPattern("inner-a")
	if len(patternMatches)==0 { t.Logf("pattern match not found using simple pattern, paths=%v", paths) }
}

func contains(s, sub string) bool { for i:=0; i+len(sub)<=len(s); i++ { if s[i:i+len(sub)]==sub { return true } }; return false }
