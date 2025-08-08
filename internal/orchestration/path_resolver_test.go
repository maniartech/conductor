package orchestration

import (
	"fmt"
	"testing"
)

// TestPathResolverBase_ParsePath tests the path parsing functionality
func TestPathResolverBase_ParsePath(t *testing.T) {
	resolver := NewPathResolverBase()

	testCases := []struct {
		path     string
		expected []string
		hasError bool
	}{
		{
			path:     "main-pipeline",
			expected: []string{"main-pipeline"},
			hasError: false,
		},
		{
			path:     "main-pipeline.auth-flow",
			expected: []string{"main-pipeline", "auth-flow"},
			hasError: false,
		},
		{
			path:     "main-pipeline.auth-flow.validate-user",
			expected: []string{"main-pipeline", "auth-flow", "validate-user"},
			hasError: false,
		},
		{
			path:     "",
			expected: nil,
			hasError: true,
		},
		{
			path:     ".",
			expected: nil,
			hasError: true,
		},
		{
			path:     "main..auth",
			expected: nil,
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			segments, err := resolver.ParsePath(tc.path)

			if tc.hasError {
				if err == nil {
					t.Errorf("Expected error for path '%s', but got none", tc.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path '%s': %v", tc.path, err)
				}
				if len(segments) != len(tc.expected) {
					t.Errorf("Expected %d segments, got %d", len(tc.expected), len(segments))
				}
				for i, expected := range tc.expected {
					if i >= len(segments) || segments[i] != expected {
						t.Errorf("Expected segment %d to be '%s', got '%s'", i, expected, segments[i])
					}
				}
			}
		})
	}
}

// TestPathResolverBase_ValidatePath tests path validation
func TestPathResolverBase_ValidatePath(t *testing.T) {
	resolver := NewPathResolverBase()

	validPaths := []string{
		"main-pipeline",
		"main-pipeline.auth-flow",
		"main-pipeline.auth-flow.validate-user",
		"step-0",
		"sequential-1.task-2",
	}

	invalidPaths := []string{
		"",
		".",
		".main",
		"main.",
		"main..auth",
		"main.auth.",
		"main pipeline", // spaces not allowed
		"main/auth",     // slashes not allowed
	}

	for _, path := range validPaths {
		t.Run("valid_"+path, func(t *testing.T) {
			if !resolver.ValidatePath(path) {
				t.Errorf("Expected path '%s' to be valid", path)
			}
		})
	}

	for _, path := range invalidPaths {
		t.Run("invalid_"+path, func(t *testing.T) {
			if resolver.ValidatePath(path) {
				t.Errorf("Expected path '%s' to be invalid", path)
			}
		})
	}
}

// TestPathResolverBase_NormalizePath tests path normalization
func TestPathResolverBase_NormalizePath(t *testing.T) {
	resolver := NewPathResolverBase()

	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "main-pipeline",
			expected: "main-pipeline",
		},
		{
			input:    "Main-Pipeline",
			expected: "main-pipeline",
		},
		{
			input:    "MAIN-PIPELINE.AUTH-FLOW",
			expected: "main-pipeline.auth-flow",
		},
		{
			input:    "main_pipeline.auth_flow",
			expected: "main-pipeline.auth-flow",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			normalized := resolver.NormalizePath(tc.input)
			if normalized != tc.expected {
				t.Errorf("Expected normalized path '%s', got '%s'", tc.expected, normalized)
			}
		})
	}
}

// TestPathMatch tests the PathMatch structure
func TestPathMatch(t *testing.T) {
	match := PathMatch{
		Path:          "main-pipeline.auth-flow.validate-user",
		Orchestration: nil, // Would be set to actual orchestration in real usage
		Depth:         2,
		Parent:        "main-pipeline.auth-flow",
		Type:          "task",
		Index:         1,
	}

	// Test GetSegments
	segments := match.GetSegments()
	expected := []string{"main-pipeline", "auth-flow", "validate-user"}

	if len(segments) != len(expected) {
		t.Errorf("Expected %d segments, got %d", len(expected), len(segments))
	}

	for i, expectedSegment := range expected {
		if i >= len(segments) || segments[i] != expectedSegment {
			t.Errorf("Expected segment %d to be '%s', got '%s'", i, expectedSegment, segments[i])
		}
	}

	// Test GetName
	name := match.GetName()
	if name != "validate-user" {
		t.Errorf("Expected name 'validate-user', got '%s'", name)
	}

	// Test IsRoot
	if match.IsRoot() {
		t.Error("Expected match not to be root")
	}

	// Test root match
	rootMatch := PathMatch{
		Path:  "main-pipeline",
		Depth: 0,
	}

	if !rootMatch.IsRoot() {
		t.Error("Expected root match to be root")
	}
}

// TestOrchestrationTree tests the tree structure
func TestOrchestrationTree(t *testing.T) {
	// Create a simple tree structure for testing
	tree := &OrchestrationTree{
		Name:  "main-pipeline",
		Path:  "main-pipeline",
		Type:  "sequential",
		Depth: 0,
		Children: []*OrchestrationTree{
			{
				Name:  "auth-flow",
				Path:  "main-pipeline.auth-flow",
				Type:  "sequential",
				Depth: 1,
				Children: []*OrchestrationTree{
					{
						Name:  "validate-user",
						Path:  "main-pipeline.auth-flow.validate-user",
						Type:  "task",
						Depth: 2,
					},
				},
			},
			{
				Name:  "cleanup",
				Path:  "main-pipeline.cleanup",
				Type:  "task",
				Depth: 1,
			},
		},
	}

	// Test GetAllPaths
	paths := tree.GetAllPaths()
	expectedPaths := []string{
		"main-pipeline",
		"main-pipeline.auth-flow",
		"main-pipeline.auth-flow.validate-user",
		"main-pipeline.cleanup",
	}

	if len(paths) != len(expectedPaths) {
		t.Errorf("Expected %d paths, got %d", len(expectedPaths), len(paths))
	}

	for i, expectedPath := range expectedPaths {
		if i >= len(paths) || paths[i] != expectedPath {
			t.Errorf("Expected path %d to be '%s', got '%s'", i, expectedPath, paths[i])
		}
	}

	// Test FindByName
	matches := tree.FindByName("auth-flow")
	if len(matches) != 1 {
		t.Errorf("Expected 1 match for 'auth-flow', got %d", len(matches))
	}
	if len(matches) > 0 && matches[0].Path != "main-pipeline.auth-flow" {
		t.Errorf("Expected match path 'main-pipeline.auth-flow', got '%s'", matches[0].Path)
	}

	// Test FindByType
	taskMatches := tree.FindByType("task")
	if len(taskMatches) != 2 {
		t.Errorf("Expected 2 task matches, got %d", len(taskMatches))
	}

	sequentialMatches := tree.FindByType("sequential")
	if len(sequentialMatches) != 2 {
		t.Errorf("Expected 2 sequential matches, got %d", len(sequentialMatches))
	}

	// Test FindByDepth
	depthMatches := tree.FindByDepth(1)
	if len(depthMatches) != 2 {
		t.Errorf("Expected 2 matches at depth 1, got %d", len(depthMatches))
	}

	// Test GetLeafNodes
	leafNodes := tree.GetLeafNodes()
	if len(leafNodes) != 2 {
		t.Errorf("Expected 2 leaf nodes, got %d", len(leafNodes))
	}
}

// TestPathQuery tests the query functionality
func TestPathQuery(t *testing.T) {
	// Create a mock tree for testing
	tree := &OrchestrationTree{
		Name:  "main-pipeline",
		Path:  "main-pipeline",
		Type:  "sequential",
		Depth: 0,
		Children: []*OrchestrationTree{
			{
				Name:  "auth-task",
				Path:  "main-pipeline.auth-task",
				Type:  "task",
				Depth: 1,
			},
			{
				Name:  "auth-pipeline",
				Path:  "main-pipeline.auth-pipeline",
				Type:  "sequential",
				Depth: 1,
				Children: []*OrchestrationTree{
					{
						Name:  "validate-auth",
						Path:  "main-pipeline.auth-pipeline.validate-auth",
						Type:  "task",
						Depth: 2,
					},
				},
			},
			{
				Name:  "data-task",
				Path:  "main-pipeline.data-task",
				Type:  "task",
				Depth: 1,
			},
		},
	}

	query := NewPathQuery(tree)

	// Test FindByPattern
	t.Run("pattern_matching", func(t *testing.T) {
		authMatches := query.FindByPattern("*auth*")
		if len(authMatches) < 2 {
			t.Errorf("Expected at least 2 matches for '*auth*', got %d", len(authMatches))
		}

		pipelineMatches := query.FindByPattern("*pipeline*")
		if len(pipelineMatches) < 2 {
			t.Errorf("Expected at least 2 matches for '*pipeline*', got %d", len(pipelineMatches))
		}
	})

	// Test FindByType
	t.Run("type_matching", func(t *testing.T) {
		taskMatches := query.FindByType("task")
		if len(taskMatches) != 3 {
			t.Errorf("Expected 3 task matches, got %d", len(taskMatches))
		}

		sequentialMatches := query.FindByType("sequential")
		if len(sequentialMatches) != 2 {
			t.Errorf("Expected 2 sequential matches, got %d", len(sequentialMatches))
		}
	})

	// Test FindByDepth
	t.Run("depth_matching", func(t *testing.T) {
		depth1Matches := query.FindByDepth(1)
		if len(depth1Matches) != 3 {
			t.Errorf("Expected 3 matches at depth 1, got %d", len(depth1Matches))
		}

		depth2Matches := query.FindByDepth(2)
		if len(depth2Matches) != 1 {
			t.Errorf("Expected 1 match at depth 2, got %d", len(depth2Matches))
		}
	})

	// Test FindLeafNodes
	t.Run("leaf_nodes", func(t *testing.T) {
		leafMatches := query.FindLeafNodes()
		if len(leafMatches) != 3 {
			t.Errorf("Expected 3 leaf nodes, got %d", len(leafMatches))
		}
	})

	// Test complex queries
	t.Run("complex_queries", func(t *testing.T) {
		// Find all tasks with "auth" in their name
		results := query.FindByPattern("*auth*")
		taskResults := make([]PathMatch, 0)
		for _, match := range results {
			if match.Type == "task" {
				taskResults = append(taskResults, match)
			}
		}

		if len(taskResults) < 2 {
			t.Errorf("Expected at least 2 auth tasks, got %d", len(taskResults))
		}
	})
}

// TestPathResolverPerformance tests the performance characteristics
func TestPathResolverPerformance(t *testing.T) {
	resolver := NewPathResolverBase()

	// Test path parsing performance
	testPath := "main-pipeline.auth-flow.validation-pipeline.validate-user.check-permissions"

	// Parse the same path multiple times to test performance
	for i := 0; i < 1000; i++ {
		segments, err := resolver.ParsePath(testPath)
		if err != nil {
			t.Fatalf("Unexpected error parsing path: %v", err)
		}
		if len(segments) != 5 {
			t.Fatalf("Expected 5 segments, got %d", len(segments))
		}
	}

	// Test path validation performance
	for i := 0; i < 1000; i++ {
		if !resolver.ValidatePath(testPath) {
			t.Fatal("Expected path to be valid")
		}
	}
}

// BenchmarkPathParsing benchmarks path parsing performance
func BenchmarkPathParsing(b *testing.B) {
	resolver := NewPathResolverBase()
	testPath := "main-pipeline.auth-flow.validation-pipeline.validate-user.check-permissions"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := resolver.ParsePath(testPath)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkPathValidation benchmarks path validation performance
func BenchmarkPathValidation(b *testing.B) {
	resolver := NewPathResolverBase()
	testPath := "main-pipeline.auth-flow.validation-pipeline.validate-user.check-permissions"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if !resolver.ValidatePath(testPath) {
			b.Fatal("Expected path to be valid")
		}
	}
}

// BenchmarkTreeTraversal benchmarks tree traversal performance
func BenchmarkTreeTraversal(b *testing.B) {
	// Create a moderately complex tree
	tree := createTestTree(3, 4) // 3 levels deep, 4 children per level

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		paths := tree.GetAllPaths()
		if len(paths) == 0 {
			b.Fatal("Expected non-empty paths")
		}
	}
}

// Helper function to create a test tree
func createTestTree(depth, childrenPerLevel int) *OrchestrationTree {
	if depth <= 0 {
		return &OrchestrationTree{
			Name:  "leaf",
			Path:  "leaf",
			Type:  "task",
			Depth: 0,
		}
	}

	children := make([]*OrchestrationTree, childrenPerLevel)
	for i := 0; i < childrenPerLevel; i++ {
		child := createTestTree(depth-1, childrenPerLevel)
		child.Name = fmt.Sprintf("child-%d", i)
		child.Path = fmt.Sprintf("root.child-%d", i)
		child.Depth = depth - 1
		children[i] = child
	}

	return &OrchestrationTree{
		Name:     "root",
		Path:     "root",
		Type:     "sequential",
		Depth:    depth,
		Children: children,
	}
}
