package sequential

import (
	"context"
	"fmt"
	"testing"

	"github.com/maniartech/orchestrator"
	"github.com/maniartech/orchestrator/internal/orchestration"
	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/errors"
)

// TestDynamicNameGeneration demonstrates the current dynamic name generation capabilities
func TestDynamicNameGeneration(t *testing.T) {
	t.Log("=== Testing Dynamic Name Generation ===")

	// Create a sequential with mixed named and unnamed tasks
	seq := Sequential(
		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "user-data", nil
		}).Named("fetch-user"), // Explicit name

		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "validated", nil
		}), // Unnamed - should generate "step-1"

		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "processed", nil
		}).Named("process-user"), // Explicit name

		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "saved", nil
		}), // Unnamed - should generate "step-3"
	).Named("user-pipeline")

	// Cast to SequentialBuilder to access dynamic methods
	seqBuilder := seq.(*SequentialBuilder)

	// Test GetChildNames functionality
	names := seqBuilder.GetChildNames()
	expectedNames := []string{"fetch-user", "step-1", "process-user", "step-3"}

	if len(names) != len(expectedNames) {
		t.Fatalf("Expected %d names, got %d", len(expectedNames), len(names))
	}

	for i, expected := range expectedNames {
		if names[i] != expected {
			t.Errorf("Expected name at index %d to be %q, got %q", i, expected, names[i])
		}
	}

	t.Logf("✅ Generated names: %v", names)

	// Test GetChildAt functionality
	childNames := seqBuilder.GetChildNames()
	for i := 0; i < seqBuilder.GetChildCount(); i++ {
		expectedName := expectedNames[i]

		actualName := childNames[i] // Use proper name access method
		if actualName != expectedName {
			t.Errorf("Child at index %d: expected name %q, got %q", i, expectedName, actualName)
		}

		t.Logf("   Child %d: %s", i, actualName)
	}

	// Test FindChildByName with both explicit and generated names
	testCases := []struct {
		name          string
		expectedIndex int
		shouldFind    bool
	}{
		{"fetch-user", 0, true},
		{"step-1", 1, true},
		{"process-user", 2, true},
		{"step-3", 3, true},
		{"nonexistent", -1, false},
	}

	for _, tc := range testCases {
		index, child := seqBuilder.FindChildByName(tc.name)
		if tc.shouldFind {
			if child == nil {
				t.Errorf("Expected to find child with name %q, but got nil", tc.name)
			} else if index != tc.expectedIndex {
				t.Errorf("Expected to find %q at index %d, but found at index %d", tc.name, tc.expectedIndex, index)
			} else {
				t.Logf("   Found %q at index %d ✅", tc.name, index)
			}
		} else {
			if child != nil {
				t.Errorf("Expected not to find child with name %q, but found at index %d", tc.name, index)
			}
		}
	}

	// Execute and test result access patterns
	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errors.FailFast}

	result, err := seq.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Test result access using generated names
	testResults := []struct {
		name     string
		expected string
	}{
		{"fetch-user", "user-data"},
		{"step-1", "validated"},
		{"process-user", "processed"},
		{"step-3", "saved"},
	}

	for _, tr := range testResults {
		actual := result.Get(tr.name)
		if actual != tr.expected {
			t.Errorf("Result for %q: expected %q, got %v", tr.name, tr.expected, actual)
		} else {
			t.Logf("   Result[%q] = %q ✅", tr.name, actual)
		}
	}
}

// TestHierarchicalNaming demonstrates hierarchical naming in nested orchestrations
func TestHierarchicalNaming(t *testing.T) {
	t.Log("=== Testing Hierarchical Naming ===")

	// Create nested sequential orchestrations
	innerSeq := Sequential(
		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "inner-result-1", nil
		}), // Should generate name based on parent context

		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "inner-result-2", nil
		}).Named("custom-inner"),
	).Named("inner-pipeline")

	outerSeq := Sequential(
		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "outer-result-1", nil
		}).Named("setup"),

		innerSeq,

		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "outer-result-3", nil
		}), // Should generate "step-2"
	).Named("outer-pipeline")

	// Cast to access dynamic methods
	outerSeqBuilder := outerSeq.(*SequentialBuilder)
	innerSeqBuilder := innerSeq.(*SequentialBuilder)

	// Test outer sequence names
	outerNames := outerSeqBuilder.GetChildNames()
	expectedOuterNames := []string{"setup", "inner-pipeline", "step-2"}

	if len(outerNames) != len(expectedOuterNames) {
		t.Fatalf("Expected %d outer names, got %d", len(expectedOuterNames), len(outerNames))
	}

	for i, expected := range expectedOuterNames {
		if outerNames[i] != expected {
			t.Errorf("Outer name at index %d: expected %q, got %q", i, expected, outerNames[i])
		}
	}

	t.Logf("✅ Outer names: %v", outerNames)

	// Test inner sequence names
	innerNames := innerSeqBuilder.GetChildNames()
	expectedInnerNames := []string{"step-0", "custom-inner"}

	if len(innerNames) != len(expectedInnerNames) {
		t.Fatalf("Expected %d inner names, got %d", len(expectedInnerNames), len(innerNames))
	}

	for i, expected := range expectedInnerNames {
		if innerNames[i] != expected {
			t.Errorf("Inner name at index %d: expected %q, got %q", i, expected, innerNames[i])
		}
	}

	t.Logf("✅ Inner names: %v", innerNames)

	// Execute and verify hierarchical result access
	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errors.FailFast}

	result, err := outerSeq.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Hierarchical execution failed: %v", err)
	}

	// Verify results are accessible by their names
	expectedResults := map[string]string{
		"setup":        "outer-result-1",
		"step-0":       "inner-result-1", // From inner sequence
		"custom-inner": "inner-result-2", // From inner sequence
		"step-2":       "outer-result-3",
	}

	for name, expected := range expectedResults {
		actual := result.Get(name)
		if actual != expected {
			t.Errorf("Hierarchical result for %q: expected %q, got %v", name, expected, actual)
		} else {
			t.Logf("   Hierarchical Result[%q] = %q ✅", name, actual)
		}
	}
}

// TestIndexBasedAccess demonstrates zero-allocation index-based access patterns
func TestIndexBasedAccess(t *testing.T) {
	t.Log("=== Testing Index-Based Access ===")

	// Create a larger sequential for index testing
	tasks := make([]orchestration.Orchestration, 10)
	for i := 0; i < 10; i++ {
		taskIndex := i // Capture loop variable
		if i%3 == 0 {
			// Every third task gets an explicit name
			tasks[i] = task.Task(func(ctx orchestrator.Context) (int, error) {
				return taskIndex * 10, nil
			}).Named(fmt.Sprintf("explicit-task-%d", taskIndex))
		} else {
			// Others get generated names
			tasks[i] = task.Task(func(ctx orchestrator.Context) (int, error) {
				return taskIndex * 10, nil
			})
		}
	}

	seq := Sequential(tasks...).Named("index-test-pipeline")
	seqBuilder := seq.(*SequentialBuilder)

	// Test GetChildCount
	count := seqBuilder.GetChildCount()
	if count != 10 {
		t.Errorf("Expected child count 10, got %d", count)
	}
	t.Logf("✅ Child count: %d", count)

	// Test GetChildAt for all positions and verify names
	childNames := seqBuilder.GetChildNames()
	for i := 0; i < count; i++ {
		child, err := seqBuilder.GetChildAt(i)
		if err != nil {
			t.Errorf("Failed to get child at index %d: %v", i, err)
			continue
		}

		if child == nil {
			t.Errorf("Child at index %d is nil", i)
			continue
		}

		// Verify the child name matches expected pattern
		expectedName := ""
		if i%3 == 0 {
			expectedName = fmt.Sprintf("explicit-task-%d", i)
		} else {
			expectedName = fmt.Sprintf("step-%d", i)
		}

		actualName := childNames[i]
		if actualName != expectedName {
			t.Errorf("Child at index %d: expected name %q, got %q", i, expectedName, actualName)
		} else {
			t.Logf("   Index %d: %s ✅", i, actualName)
		}
	}

	// Test bounds checking
	_, err := seqBuilder.GetChildAt(-1)
	if err == nil {
		t.Error("Expected error for negative index, got nil")
	}

	_, err = seqBuilder.GetChildAt(count)
	if err == nil {
		t.Error("Expected error for index >= count, got nil")
	}

	_, err = seqBuilder.GetChildAt(count + 10)
	if err == nil {
		t.Error("Expected error for index >> count, got nil")
	}

	t.Logf("✅ Bounds checking works correctly")
}

// TestNameConsistencyAcrossExecution verifies names remain consistent during execution
func TestNameConsistencyAcrossExecution(t *testing.T) {
	t.Log("=== Testing Name Consistency Across Execution ===")

	seq := Sequential(
		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "step1-result", nil
		}), // step-0

		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "step2-result", nil
		}).Named("named-step"),

		task.Task(func(ctx orchestrator.Context) (string, error) {
			return "step3-result", nil
		}), // step-2
	).Named("consistency-test")

	seqBuilder := seq.(*SequentialBuilder)

	// Get names before execution
	namesBefore := seqBuilder.GetChildNames()
	t.Logf("Names before execution: %v", namesBefore)

	// Execute
	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errors.FailFast}

	result, err := seq.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Get names after execution
	namesAfter := seqBuilder.GetChildNames()
	t.Logf("Names after execution: %v", namesAfter)

	// Verify names are consistent
	if len(namesBefore) != len(namesAfter) {
		t.Errorf("Name count changed: before=%d, after=%d", len(namesBefore), len(namesAfter))
	}

	for i, nameBefore := range namesBefore {
		if i < len(namesAfter) && namesBefore[i] != namesAfter[i] {
			t.Errorf("Name at index %d changed: before=%q, after=%q", i, nameBefore, namesAfter[i])
		}
	}

	// Verify results are accessible by the same names
	for _, name := range namesBefore {
		if result.Get(name) == nil {
			t.Errorf("Result not accessible by name %q after execution", name)
		}
	}

	t.Logf("✅ Names remained consistent across execution")
}
