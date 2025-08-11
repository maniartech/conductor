package types

import (
	"sync"
	"testing"
)

// TestNewRootNamingContext tests root naming context creation
func TestNewRootNamingContext(t *testing.T) {
	tests := []struct {
		name              string
		rootName          string
		orchestrationType string
		expectedPath      string
		expectedDepth     int
	}{
		{
			name:              "simple_root",
			rootName:          "main",
			orchestrationType: "sequential",
			expectedPath:      "main",
			expectedDepth:     0,
		},
		{
			name:              "empty_root_name",
			rootName:          "",
			orchestrationType: "task",
			expectedPath:      "",
			expectedDepth:     0,
		},
		{
			name:              "complex_root_name",
			rootName:          "user-authentication-pipeline",
			orchestrationType: "concurrent",
			expectedPath:      "user-authentication-pipeline",
			expectedDepth:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nc := NewRootNamingContext(tt.rootName, tt.orchestrationType)

			if nc == nil {
				t.Fatal("Expected naming context to be created")
			}

			if nc.GetPath() != tt.expectedPath {
				t.Errorf("Expected path '%s', got '%s'", tt.expectedPath, nc.GetPath())
			}

			if nc.GetDepth() != tt.expectedDepth {
				t.Errorf("Expected depth %d, got %d", tt.expectedDepth, nc.GetDepth())
			}

			if nc.GetName() != tt.rootName {
				t.Errorf("Expected name '%s', got '%s'", tt.rootName, nc.GetName())
			}

			if nc.GetParent() != nil {
				t.Error("Expected root context to have no parent")
			}

			if nc.orchestrationType != tt.orchestrationType {
				t.Errorf("Expected orchestration type '%s', got '%s'", tt.orchestrationType, nc.orchestrationType)
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
		expectedPath      string
		expectedDepth     int
	}{
		{
			name:              "simple_child",
			childName:         "auth",
			orchestrationType: "task",
			expectedPath:      "main.auth",
			expectedDepth:     1,
		},
		{
			name:              "nested_child",
			childName:         "validate-user",
			orchestrationType: "sequential",
			expectedPath:      "main.validate-user",
			expectedDepth:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			child := NewChildNamingContext(parent, tt.childName, tt.orchestrationType)

			if child == nil {
				t.Fatal("Expected child naming context to be created")
			}

			if child.GetPath() != tt.expectedPath {
				t.Errorf("Expected path '%s', got '%s'", tt.expectedPath, child.GetPath())
			}

			if child.GetDepth() != tt.expectedDepth {
				t.Errorf("Expected depth %d, got %d", tt.expectedDepth, child.GetDepth())
			}

			if child.GetName() != tt.childName {
				t.Errorf("Expected name '%s', got '%s'", tt.childName, child.GetName())
			}

			if child.GetParent() != parent {
				t.Error("Expected child to have correct parent")
			}

			if child.GetParentPath() != parent.GetPath() {
				t.Errorf("Expected parent path '%s', got '%s'", parent.GetPath(), child.GetParentPath())
			}
		})
	}
}

// TestNamingContext_GenerateChildName tests child name generation
func TestNamingContext_GenerateChildName(t *testing.T) {
	nc := NewRootNamingContext("main", "sequential")

	tests := []struct {
		name              string
		childName         string
		orchestrationType string
		index             int
		expectedName      string
		isNamed           bool
	}{
		{
			name:              "explicit_name",
			childName:         "auth-task",
			orchestrationType: "task",
			index:             0,
			expectedName:      "auth-task",
			isNamed:           true,
		},
		{
			name:              "generated_task_name",
			childName:         "",
			orchestrationType: "task",
			index:             0,
			expectedName:      "task-1",
			isNamed:           false,
		},
		{
			name:              "generated_sequential_name",
			childName:         "",
			orchestrationType: "sequential",
			index:             0,
			expectedName:      "seq-1",
			isNamed:           false,
		},
		{
			name:              "generated_concurrent_name",
			childName:         "",
			orchestrationType: "concurrent",
			index:             0,
			expectedName:      "conc-1",
			isNamed:           false,
		},
		{
			name:              "generated_default_name",
			childName:         "",
			orchestrationType: "unknown",
			index:             0,
			expectedName:      "step-1",
			isNamed:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generatedName := nc.GenerateChildName(tt.childName, tt.orchestrationType, tt.index)

			if generatedName != tt.expectedName {
				t.Errorf("Expected name '%s', got '%s'", tt.expectedName, generatedName)
			}

			if nc.IsNamed(generatedName) != tt.isNamed {
				t.Errorf("Expected IsNamed to be %v for name '%s'", tt.isNamed, generatedName)
			}
		})
	}
}

// TestNamingContext_GetRoot tests root context retrieval
func TestNamingContext_GetRoot(t *testing.T) {
	root := NewRootNamingContext("main", "sequential")
	child1 := NewChildNamingContext(root, "auth", "task")
	child2 := NewChildNamingContext(child1, "validate", "task")

	// Test root returns itself
	if root.GetRoot() != root {
		t.Error("Expected root to return itself")
	}

	// Test child returns root
	if child1.GetRoot() != root {
		t.Error("Expected child1 to return root")
	}

	// Test grandchild returns root
	if child2.GetRoot() != root {
		t.Error("Expected child2 to return root")
	}
}

// TestNamingContext_ConcurrentAccess tests thread safety
func TestNamingContext_ConcurrentAccess(t *testing.T) {
	nc := NewRootNamingContext("main", "sequential")

	var wg sync.WaitGroup
	const numGoroutines = 100

	// Test concurrent name generation
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			name := nc.GenerateChildName("", "task", index)
			if name == "" {
				t.Errorf("Expected non-empty name for index %d", index)
			}
		}(i)
	}

	// Test concurrent path access
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			path := nc.GetPath()
			if path != "main" {
				t.Errorf("Expected path 'main', got '%s'", path)
			}
		}()
	}

	wg.Wait()
}

// TestHierarchicalNamer tests hierarchical namer functionality
func TestHierarchicalNamer(t *testing.T) {
	t.Run("root_namer", func(t *testing.T) {
		hn := NewHierarchicalNamer(nil, "main", "sequential", 0)

		if hn.GetOperationID() != "main" {
			t.Errorf("Expected operation ID 'main', got '%s'", hn.GetOperationID())
		}

		if hn.GetName() != "main" {
			t.Errorf("Expected name 'main', got '%s'", hn.GetName())
		}

		if !hn.IsRoot() {
			t.Error("Expected root namer to be root")
		}

		if hn.GetDepth() != 0 {
			t.Errorf("Expected depth 0, got %d", hn.GetDepth())
		}

		if hn.GetParentPath() != "" {
			t.Errorf("Expected empty parent path, got '%s'", hn.GetParentPath())
		}
	})

	t.Run("child_namer", func(t *testing.T) {
		parent := NewHierarchicalNamer(nil, "main", "sequential", 0)
		child := NewHierarchicalNamer(parent.GetContext(), "auth", "task", 0)

		if child.GetOperationID() != "main.auth" {
			t.Errorf("Expected operation ID 'main.auth', got '%s'", child.GetOperationID())
		}

		if child.GetName() != "auth" {
			t.Errorf("Expected name 'auth', got '%s'", child.GetName())
		}

		if child.IsRoot() {
			t.Error("Expected child namer not to be root")
		}

		if child.GetDepth() != 1 {
			t.Errorf("Expected depth 1, got %d", child.GetDepth())
		}

		if child.GetParentPath() != "main" {
			t.Errorf("Expected parent path 'main', got '%s'", child.GetParentPath())
		}
	})

	t.Run("create_child_namer", func(t *testing.T) {
		parent := NewHierarchicalNamer(nil, "main", "sequential", 0)
		child := parent.CreateChildNamer("auth", "task", 0)

		if child.GetOperationID() != "main.auth" {
			t.Errorf("Expected operation ID 'main.auth', got '%s'", child.GetOperationID())
		}

		if child.GetParentPath() != "main" {
			t.Errorf("Expected parent path 'main', got '%s'", child.GetParentPath())
		}
	})

	t.Run("path_segments", func(t *testing.T) {
		parent := NewHierarchicalNamer(nil, "main", "sequential", 0)
		child := NewHierarchicalNamer(parent.GetContext(), "auth", "task", 0)
		grandchild := NewHierarchicalNamer(child.GetContext(), "validate", "task", 1)

		segments := grandchild.GetPathSegments()
		expected := []string{"main", "auth", "validate"}

		if len(segments) != len(expected) {
			t.Errorf("Expected %d segments, got %d", len(expected), len(segments))
		}

		for i, segment := range segments {
			if segment != expected[i] {
				t.Errorf("Expected segment %d to be '%s', got '%s'", i, expected[i], segment)
			}
		}
	})

	t.Run("unnamed_root", func(t *testing.T) {
		hn := NewHierarchicalNamer(nil, "", "sequential", 0)

		if hn.GetName() != "root" {
			t.Errorf("Expected name 'root' for unnamed root, got '%s'", hn.GetName())
		}

		if hn.GetOperationID() != "root" {
			t.Errorf("Expected operation ID 'root', got '%s'", hn.GetOperationID())
		}
	})

	t.Run("type_and_index", func(t *testing.T) {
		hn := NewHierarchicalNamer(nil, "main", "sequential", 5)

		if hn.GetType() != "sequential" {
			t.Errorf("Expected type 'sequential', got '%s'", hn.GetType())
		}

		if hn.GetIndex() != 5 {
			t.Errorf("Expected index 5, got %d", hn.GetIndex())
		}
	})
}

// TestHierarchicalNamer_IsNamed tests name detection
func TestHierarchicalNamer_IsNamed(t *testing.T) {
	parent := NewHierarchicalNamer(nil, "main", "sequential", 0)

	tests := []struct {
		name          string
		childName     string
		expectedNamed bool
	}{
		{
			name:          "explicit_name",
			childName:     "auth-task",
			expectedNamed: true,
		},
		{
			name:          "empty_name",
			childName:     "",
			expectedNamed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			child := NewHierarchicalNamer(parent.GetContext(), tt.childName, "task", 0)

			if child.IsNamed() != tt.expectedNamed {
				t.Errorf("Expected IsNamed to be %v for name '%s'", tt.expectedNamed, tt.childName)
			}
		})
	}
}

// BenchmarkNamingContext_GenerateChildName benchmarks name generation
func BenchmarkNamingContext_GenerateChildName(b *testing.B) {
	nc := NewRootNamingContext("main", "sequential")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nc.GenerateChildName("", "task", i)
	}
}

// BenchmarkNamingContext_GetPath benchmarks path retrieval
func BenchmarkNamingContext_GetPath(b *testing.B) {
	nc := NewRootNamingContext("main", "sequential")
	child := NewChildNamingContext(nc, "auth", "task")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		child.GetPath()
	}
}

// BenchmarkHierarchicalNamer_GetOperationID benchmarks operation ID retrieval
func BenchmarkHierarchicalNamer_GetOperationID(b *testing.B) {
	parent := NewHierarchicalNamer(nil, "main", "sequential", 0)
	child := NewHierarchicalNamer(parent.GetContext(), "auth", "task", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		child.GetOperationID()
	}
}
