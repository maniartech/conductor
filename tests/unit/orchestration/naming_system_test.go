package orchestration

import (
	"testing"

	"github.com/maniartech/orchestrator/pkg/types"

	. "github.com/maniartech/orchestrator/internal/orchestration"
)

// TestNewRootNamingContext tests root naming context creation
func TestNewRootNamingContext(t *testing.T) {
	tests := []struct {
		name              string
		rootName          string
		orchestrationType string
	}{
		{"simple_root", "main", "sequential"},
		{"empty_name", "", "task"},
		{"complex_name", "user-auth-pipeline", "concurrent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nc := NewRootNamingContext(tt.rootName, tt.orchestrationType)

			if nc == nil {
				t.Fatal("Expected naming context to be created")
			}

			// Verify it's the same as calling types.NewRootNamingContext
			expected := types.NewRootNamingContext(tt.rootName, tt.orchestrationType)

			if nc.GetPath() != expected.GetPath() {
				t.Errorf("Expected path '%s', got '%s'", expected.GetPath(), nc.GetPath())
			}

			if nc.GetName() != expected.GetName() {
				t.Errorf("Expected name '%s', got '%s'", expected.GetName(), nc.GetName())
			}

			if nc.GetDepth() != expected.GetDepth() {
				t.Errorf("Expected depth %d, got %d", expected.GetDepth(), nc.GetDepth())
			}
		})
	}
}

// TestNewChildNamingContext tests child naming context creation
func TestNewChildNamingContext(t *testing.T) {
	parent := NewRootNamingContext("main", "sequential")

	tests := []struct {
		name              string
		childName         string
		orchestrationType string
	}{
		{"simple_child", "auth", "task"},
		{"complex_child", "validate-user", "sequential"},
		{"empty_child", "", "concurrent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			child := NewChildNamingContext(parent, tt.childName, tt.orchestrationType)

			if child == nil {
				t.Fatal("Expected child naming context to be created")
			}

			// Verify it's the same as calling types.NewChildNamingContext
			expected := types.NewChildNamingContext(parent, tt.childName, tt.orchestrationType)

			if child.GetPath() != expected.GetPath() {
				t.Errorf("Expected path '%s', got '%s'", expected.GetPath(), child.GetPath())
			}

			if child.GetName() != expected.GetName() {
				t.Errorf("Expected name '%s', got '%s'", expected.GetName(), child.GetName())
			}

			if child.GetDepth() != expected.GetDepth() {
				t.Errorf("Expected depth %d, got %d", expected.GetDepth(), child.GetDepth())
			}

			if child.GetParent() != parent {
				t.Error("Expected child to have correct parent")
			}
		})
	}
}

// TestNewHierarchicalNamer tests hierarchical namer creation
func TestNewHierarchicalNamer(t *testing.T) {
	parent := NewRootNamingContext("main", "sequential")

	tests := []struct {
		name              string
		parentContext     *types.NamingContext
		childName         string
		orchestrationType string
		index             int
	}{
		{"root_namer", nil, "root", "sequential", 0},
		{"child_namer", parent, "auth", "task", 1},
		{"unnamed_child", parent, "", "concurrent", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namer := NewHierarchicalNamer(tt.parentContext, tt.childName, tt.orchestrationType, tt.index)

			if namer == nil {
				t.Fatal("Expected hierarchical namer to be created")
			}

			// Verify it's the same as calling types.NewHierarchicalNamer
			expected := types.NewHierarchicalNamer(tt.parentContext, tt.childName, tt.orchestrationType, tt.index)

			if namer.GetOperationID() != expected.GetOperationID() {
				t.Errorf("Expected operation ID '%s', got '%s'", expected.GetOperationID(), namer.GetOperationID())
			}

			if namer.GetName() != expected.GetName() {
				t.Errorf("Expected name '%s', got '%s'", expected.GetName(), namer.GetName())
			}

			if namer.GetDepth() != expected.GetDepth() {
				t.Errorf("Expected depth %d, got %d", expected.GetDepth(), namer.GetDepth())
			}

			if namer.GetType() != expected.GetType() {
				t.Errorf("Expected type '%s', got '%s'", expected.GetType(), namer.GetType())
			}

			if namer.GetIndex() != expected.GetIndex() {
				t.Errorf("Expected index %d, got %d", expected.GetIndex(), namer.GetIndex())
			}
		})
	}
}

// TestNamingSystemAliases tests that aliases work correctly
func TestNamingSystemAliases(t *testing.T) {
	t.Run("naming_context_alias", func(t *testing.T) {
		// Test that NamingContext is properly aliased
		nc := NewRootNamingContext("test", "task")
		var typesNC *types.NamingContext = nc

		if typesNC == nil {
			t.Error("Expected NamingContext alias to work")
		}
	})

	t.Run("hierarchical_namer_alias", func(t *testing.T) {
		// Test that HierarchicalNamer is properly aliased
		hn := NewHierarchicalNamer(nil, "test", "task", 0)
		var typesHN *types.HierarchicalNamer = hn

		if typesHN == nil {
			t.Error("Expected HierarchicalNamer alias to work")
		}
	})
}

// TestNamingSystemIntegration tests integration between naming components
func TestNamingSystemIntegration(t *testing.T) {
	// Create a hierarchical structure
	root := NewRootNamingContext("main-pipeline", "sequential")
	child1 := NewChildNamingContext(root, "auth-flow", "sequential")
	child2 := NewChildNamingContext(child1, "validate-user", "task")

	// Create hierarchical namers
	rootNamer := NewHierarchicalNamer(nil, "main-pipeline", "sequential", 0)
	childNamer := NewHierarchicalNamer(rootNamer.GetContext(), "auth-flow", "sequential", 0)
	grandchildNamer := NewHierarchicalNamer(childNamer.GetContext(), "validate-user", "task", 0)

	t.Run("path_consistency", func(t *testing.T) {
		// Verify paths are consistent
		if child2.GetPath() != "main-pipeline.auth-flow.validate-user" {
			t.Errorf("Expected path 'main-pipeline.auth-flow.validate-user', got '%s'", child2.GetPath())
		}

		if grandchildNamer.GetOperationID() != "main-pipeline.auth-flow.validate-user" {
			t.Errorf("Expected operation ID 'main-pipeline.auth-flow.validate-user', got '%s'", grandchildNamer.GetOperationID())
		}
	})

	t.Run("depth_consistency", func(t *testing.T) {
		if root.GetDepth() != 0 {
			t.Errorf("Expected root depth 0, got %d", root.GetDepth())
		}

		if child1.GetDepth() != 1 {
			t.Errorf("Expected child1 depth 1, got %d", child1.GetDepth())
		}

		if child2.GetDepth() != 2 {
			t.Errorf("Expected child2 depth 2, got %d", child2.GetDepth())
		}

		if grandchildNamer.GetDepth() != 2 {
			t.Errorf("Expected grandchild namer depth 2, got %d", grandchildNamer.GetDepth())
		}
	})

	t.Run("parent_relationships", func(t *testing.T) {
		if child1.GetParent() != root {
			t.Error("Expected child1 parent to be root")
		}

		if child2.GetParent() != child1 {
			t.Error("Expected child2 parent to be child1")
		}

		if child2.GetRoot() != root {
			t.Error("Expected child2 root to be root")
		}
	})
}

// BenchmarkNewRootNamingContext benchmarks root context creation
func BenchmarkNewRootNamingContext(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewRootNamingContext("benchmark-root", "task")
	}
}

// BenchmarkNewChildNamingContext benchmarks child context creation
func BenchmarkNewChildNamingContext(b *testing.B) {
	parent := NewRootNamingContext("parent", "sequential")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewChildNamingContext(parent, "child", "task")
	}
}

// BenchmarkNewHierarchicalNamer benchmarks hierarchical namer creation
func BenchmarkNewHierarchicalNamer(b *testing.B) {
	parent := NewRootNamingContext("parent", "sequential")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewHierarchicalNamer(parent, "child", "task", i)
	}
}
