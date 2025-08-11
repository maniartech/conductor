package types

import (
	"testing"
)

// TestPathMatch tests PathMatch functionality
func TestPathMatch(t *testing.T) {
	t.Run("get_segments", func(t *testing.T) {
		tests := []struct {
			path     string
			expected []string
		}{
			{"", []string{}},
			{"main", []string{"main"}},
			{"main.auth", []string{"main", "auth"}},
			{"main.auth.validate", []string{"main", "auth", "validate"}},
		}

		for _, tt := range tests {
			pm := &PathMatch{Path: tt.path}
			segments := pm.GetSegments()

			if len(segments) != len(tt.expected) {
				t.Errorf("Expected %d segments, got %d", len(tt.expected), len(segments))
				continue
			}

			for i, segment := range segments {
				if segment != tt.expected[i] {
					t.Errorf("Expected segment %d to be '%s', got '%s'", i, tt.expected[i], segment)
				}
			}
		}
	})

	t.Run("get_name", func(t *testing.T) {
		tests := []struct {
			path     string
			expected string
		}{
			{"", ""},
			{"main", "main"},
			{"main.auth", "auth"},
			{"main.auth.validate", "validate"},
		}

		for _, tt := range tests {
			pm := &PathMatch{Path: tt.path}
			name := pm.GetName()

			if name != tt.expected {
				t.Errorf("Expected name '%s', got '%s'", tt.expected, name)
			}
		}
	})

	t.Run("is_root", func(t *testing.T) {
		tests := []struct {
			depth    int
			expected bool
		}{
			{0, true},
			{1, false},
			{2, false},
		}

		for _, tt := range tests {
			pm := &PathMatch{Depth: tt.depth}
			isRoot := pm.IsRoot()

			if isRoot != tt.expected {
				t.Errorf("Expected IsRoot to be %v for depth %d", tt.expected, tt.depth)
			}
		}
	})
}

// TestOrchestrationTree tests OrchestrationTree functionality
func TestOrchestrationTree(t *testing.T) {
	// Create a test tree structure
	root := &OrchestrationTree{
		Name:  "main",
		Path:  "main",
		Type:  "sequential",
		Depth: 0,
	}

	auth := &OrchestrationTree{
		Name:   "auth",
		Path:   "main.auth",
		Type:   "task",
		Depth:  1,
		Parent: root,
	}

	validate := &OrchestrationTree{
		Name:   "validate",
		Path:   "main.auth.validate",
		Type:   "task",
		Depth:  2,
		Parent: auth,
	}

	process := &OrchestrationTree{
		Name:   "process",
		Path:   "main.process",
		Type:   "concurrent",
		Depth:  1,
		Parent: root,
	}

	auth.Children = []*OrchestrationTree{validate}
	root.Children = []*OrchestrationTree{auth, process}

	t.Run("get_all_paths", func(t *testing.T) {
		paths := root.GetAllPaths()
		expected := []string{"main", "main.auth", "main.auth.validate", "main.process"}

		if len(paths) != len(expected) {
			t.Errorf("Expected %d paths, got %d", len(expected), len(paths))
		}

		for i, path := range paths {
			if path != expected[i] {
				t.Errorf("Expected path %d to be '%s', got '%s'", i, expected[i], path)
			}
		}
	})

	t.Run("find_by_name", func(t *testing.T) {
		matches := root.FindByName("auth")

		if len(matches) != 1 {
			t.Errorf("Expected 1 match, got %d", len(matches))
		}

		if matches[0].Path != "main.auth" {
			t.Errorf("Expected path 'main.auth', got '%s'", matches[0].Path)
		}

		if matches[0].Type != "task" {
			t.Errorf("Expected type 'task', got '%s'", matches[0].Type)
		}
	})

	t.Run("find_by_type", func(t *testing.T) {
		matches := root.FindByType("task")

		if len(matches) != 2 {
			t.Errorf("Expected 2 matches, got %d", len(matches))
		}

		expectedPaths := []string{"main.auth", "main.auth.validate"}
		for i, match := range matches {
			if match.Path != expectedPaths[i] {
				t.Errorf("Expected path '%s', got '%s'", expectedPaths[i], match.Path)
			}
		}
	})

	t.Run("find_by_depth", func(t *testing.T) {
		matches := root.FindByDepth(1)

		if len(matches) != 2 {
			t.Errorf("Expected 2 matches, got %d", len(matches))
		}

		expectedPaths := []string{"main.auth", "main.process"}
		for i, match := range matches {
			if match.Path != expectedPaths[i] {
				t.Errorf("Expected path '%s', got '%s'", expectedPaths[i], match.Path)
			}
		}
	})

	t.Run("get_leaf_nodes", func(t *testing.T) {
		matches := root.GetLeafNodes()

		if len(matches) != 2 {
			t.Errorf("Expected 2 leaf nodes, got %d", len(matches))
		}

		expectedPaths := []string{"main.auth.validate", "main.process"}
		for i, match := range matches {
			if match.Path != expectedPaths[i] {
				t.Errorf("Expected leaf path '%s', got '%s'", expectedPaths[i], match.Path)
			}
		}
	})

	t.Run("get_parent_path", func(t *testing.T) {
		if root.getParentPath() != "" {
			t.Errorf("Expected empty parent path for root, got '%s'", root.getParentPath())
		}

		if auth.getParentPath() != "main" {
			t.Errorf("Expected parent path 'main', got '%s'", auth.getParentPath())
		}

		if validate.getParentPath() != "main.auth" {
			t.Errorf("Expected parent path 'main.auth', got '%s'", validate.getParentPath())
		}
	})
}

// TestPathQuery tests PathQuery functionality
func TestPathQuery(t *testing.T) {
	// Create a test tree structure
	root := &OrchestrationTree{
		Name:  "main",
		Path:  "main",
		Type:  "sequential",
		Depth: 0,
	}

	auth := &OrchestrationTree{
		Name:   "auth",
		Path:   "main.auth",
		Type:   "task",
		Depth:  1,
		Parent: root,
	}

	validate := &OrchestrationTree{
		Name:   "validate",
		Path:   "main.auth.validate",
		Type:   "task",
		Depth:  2,
		Parent: auth,
	}

	process := &OrchestrationTree{
		Name:   "process",
		Path:   "main.process",
		Type:   "concurrent",
		Depth:  1,
		Parent: root,
	}

	auth.Children = []*OrchestrationTree{validate}
	root.Children = []*OrchestrationTree{auth, process}

	query := NewPathQuery(root)

	t.Run("find_by_pattern", func(t *testing.T) {
		tests := []struct {
			pattern  string
			expected []string
		}{
			{"*", []string{"main", "main.auth", "main.auth.validate", "main.process"}},
			{"**", []string{"main", "main.auth", "main.auth.validate", "main.process"}},
			{"main", []string{"main"}},
			{"*auth*", []string{"main.auth", "main.auth.validate"}},
			{"*validate*", []string{"main.auth.validate"}},
		}

		for _, tt := range tests {
			matches := query.FindByPattern(tt.pattern)

			if len(matches) != len(tt.expected) {
				t.Errorf("Pattern '%s': expected %d matches, got %d", tt.pattern, len(tt.expected), len(matches))
				continue
			}

			for i, match := range matches {
				if match.Path != tt.expected[i] {
					t.Errorf("Pattern '%s': expected path '%s', got '%s'", tt.pattern, tt.expected[i], match.Path)
				}
			}
		}
	})

	t.Run("find_by_type", func(t *testing.T) {
		matches := query.FindByType("task")

		if len(matches) != 2 {
			t.Errorf("Expected 2 task matches, got %d", len(matches))
		}

		expectedPaths := []string{"main.auth", "main.auth.validate"}
		for i, match := range matches {
			if match.Path != expectedPaths[i] {
				t.Errorf("Expected path '%s', got '%s'", expectedPaths[i], match.Path)
			}
		}
	})

	t.Run("find_by_depth", func(t *testing.T) {
		matches := query.FindByDepth(1)

		if len(matches) != 2 {
			t.Errorf("Expected 2 depth-1 matches, got %d", len(matches))
		}

		expectedPaths := []string{"main.auth", "main.process"}
		for i, match := range matches {
			if match.Path != expectedPaths[i] {
				t.Errorf("Expected path '%s', got '%s'", expectedPaths[i], match.Path)
			}
		}
	})

	t.Run("find_leaf_nodes", func(t *testing.T) {
		matches := query.FindLeafNodes()

		if len(matches) != 2 {
			t.Errorf("Expected 2 leaf nodes, got %d", len(matches))
		}

		expectedPaths := []string{"main.auth.validate", "main.process"}
		for i, match := range matches {
			if match.Path != expectedPaths[i] {
				t.Errorf("Expected leaf path '%s', got '%s'", expectedPaths[i], match.Path)
			}
		}
	})
}

// TestPathQuery_MatchesPattern tests pattern matching logic
func TestPathQuery_MatchesPattern(t *testing.T) {
	query := &PathQuery{}

	tests := []struct {
		path     string
		pattern  string
		expected bool
	}{
		{"main", "*", true},
		{"main", "**", true},
		{"main.auth", "*auth*", true},
		{"main.auth", "*process*", false},
		{"main.auth.validate", "*validate*", true},
		{"main.auth.validate", "*auth*", true},
		{"main.process", "main.process", true},
		{"main.process", "main.auth", false},
		{"", "*", true},
		{"", "**", true},
	}

	for _, tt := range tests {
		result := query.matchesPattern(tt.path, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchesPattern('%s', '%s') = %v, want %v", tt.path, tt.pattern, result, tt.expected)
		}
	}
}

// TestNewPathQuery tests PathQuery creation
func TestNewPathQuery(t *testing.T) {
	tree := &OrchestrationTree{
		Name: "test",
		Path: "test",
		Type: "sequential",
	}

	query := NewPathQuery(tree)

	if query == nil {
		t.Fatal("Expected PathQuery to be created")
	}

	if query.tree != tree {
		t.Error("Expected PathQuery to reference the correct tree")
	}
}

// TestOrchestrationTree_Print tests tree printing (coverage)
func TestOrchestrationTree_Print(t *testing.T) {
	root := &OrchestrationTree{
		Name:  "main",
		Path:  "main",
		Type:  "sequential",
		Depth: 0,
	}

	child := &OrchestrationTree{
		Name:   "auth",
		Path:   "main.auth",
		Type:   "task",
		Depth:  1,
		Parent: root,
	}

	root.Children = []*OrchestrationTree{child}

	// This test just ensures the Print method doesn't panic
	// In a real test environment, you might capture stdout to verify output
	root.Print()
}

// BenchmarkPathQuery_FindByPattern benchmarks pattern matching
func BenchmarkPathQuery_FindByPattern(b *testing.B) {
	// Create a larger tree for benchmarking
	root := &OrchestrationTree{
		Name:  "main",
		Path:  "main",
		Type:  "sequential",
		Depth: 0,
	}

	// Add many children
	for i := 0; i < 100; i++ {
		child := &OrchestrationTree{
			Name:   "task",
			Path:   "main.task",
			Type:   "task",
			Depth:  1,
			Parent: root,
		}
		root.Children = append(root.Children, child)
	}

	query := NewPathQuery(root)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.FindByPattern("*task*")
	}
}

// BenchmarkOrchestrationTree_GetAllPaths benchmarks path collection
func BenchmarkOrchestrationTree_GetAllPaths(b *testing.B) {
	// Create a larger tree for benchmarking
	root := &OrchestrationTree{
		Name:  "main",
		Path:  "main",
		Type:  "sequential",
		Depth: 0,
	}

	// Add many children
	for i := 0; i < 100; i++ {
		child := &OrchestrationTree{
			Name:   "task",
			Path:   "main.task",
			Type:   "task",
			Depth:  1,
			Parent: root,
		}
		root.Children = append(root.Children, child)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.GetAllPaths()
	}
}
