package sequential

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/internal/task"
)

// TestSequential_Constructor tests the Sequential constructor function
func TestSequential_Constructor(t *testing.T) {
	// Test valid construction
	task1 := task.Task(func() (string, error) { return "test", nil })
	task2 := task.Task(func() (int, error) { return 42, nil })

	seq := Sequential(task1, task2)

	if seq == nil {
		t.Error("Expected non-nil SequentialBuilder")
	}
	if len(seq.orchestrations) != 2 {
		t.Errorf("Expected 2 orchestrations, got %d", len(seq.orchestrations))
	}
	if seq.GetStatus() != orchestration.NotStarted {
		t.Errorf("Expected status NotStarted, got %v", seq.GetStatus())
	}
}

// TestSequentialBuilder_FluentAPI tests the fluent API methods
func TestSequentialBuilder_FluentAPI(t *testing.T) {
	task1 := task.Task(func() (string, error) { return "test", nil })
	task2 := task.Task(func() (int, error) { return 42, nil })

	seq := Sequential(task1, task2)

	// Test Named method
	namedSeq := seq.Named("test-sequential")
	if namedSeq != seq {
		t.Error("Named should return the same instance")
	}
	if seq.GetName() != "test-sequential" {
		t.Errorf("Expected name 'test-sequential', got %q", seq.GetName())
	}

	// Test With method
	testConfig := config.Config{
		Timeout:        30 * time.Second,
		ErrorStrategy:  errors.FailFast,
		MaxConcurrency: 50,
	}
	configuredSeq := seq.With(testConfig)
	if configuredSeq != seq {
		t.Error("With should return the same instance")
	}
	if seq.GetConfig() == nil {
		t.Error("Expected non-nil config")
	}
	if seq.GetConfig().Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", seq.GetConfig().Timeout)
	}

	// Test ErrorBoundary method
	boundarySeq := seq.ErrorBoundary(errors.CollectAll)
	if boundarySeq != seq {
		t.Error("ErrorBoundary should return the same instance")
	}
	if seq.errorBoundary == nil || *seq.errorBoundary != errors.CollectAll {
		t.Error("Expected error boundary to be set to CollectAll")
	}
}

// TestSequentialBuilder_Execute_FailFast tests fail-fast execution strategy
func TestSequentialBuilder_Execute_FailFast(t *testing.T) {
	// Test successful execution
	seq := Sequential(
		task.Task(func() (string, error) { return "result1", nil }).Named("task1"),
		task.Task(func() (int, error) { return 42, nil }).Named("task2"),
		task.Task(func() (bool, error) { return true, nil }).Named("task3"),
	)

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errors.FailFast}

	result, err := seq.Execute(ctx, cfg)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check all results are present
	if result.Get("task1") == nil {
		t.Error("Expected result from task1")
	}
	if result.Get("task2") == nil {
		t.Error("Expected result from task2")
	}
	if result.Get("task3") == nil {
		t.Error("Expected result from task3")
	}

	// Check status
	if seq.GetStatus() != orchestration.Completed {
		t.Errorf("Expected status Completed, got %v", seq.GetStatus())
	}
}

// TestSequentialBuilder_Execute_WithError tests error handling
func TestSequentialBuilder_Execute_WithError(t *testing.T) {
	// Test with first task failing (FailFast)
	seq := Sequential(
		task.Task(func() (string, error) { return "", stderrors.New("task1 failed") }).Named("task1"),
		task.Task(func() (int, error) { return 42, nil }).Named("task2"),
		task.Task(func() (bool, error) { return true, nil }).Named("task3"),
	)

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errors.FailFast}

	result, err := seq.Execute(ctx, cfg)

	if err == nil {
		t.Error("Expected error but got none")
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Should have no successful results (FailFast stopped on first error)
	if result.Get("task1") != nil {
		t.Error("Expected no result from failed task1")
	}
	if result.Get("task2") != nil {
		t.Error("Expected no result from task2 (should not have executed)")
	}
	if result.Get("task3") != nil {
		t.Error("Expected no result from task3 (should not have executed)")
	}

	// Should have one error
	if len(result.Errors()) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors()))
	}
}

// TestSequentialBuilder_Execute_CollectAll tests collect-all execution strategy
func TestSequentialBuilder_Execute_CollectAll(t *testing.T) {
	seq := Sequential(
		task.Task(func() (string, error) { return "result1", nil }).Named("task1"),
		task.Task(func() (int, error) { return 0, stderrors.New("task2 failed") }).Named("task2"),
		task.Task(func() (bool, error) { return true, nil }).Named("task3"),
	)

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errors.CollectAll}

	result, err := seq.Execute(ctx, cfg)

	// Should continue execution despite error (CollectAll behavior)
	if err == nil {
		t.Error("Expected error but got none")
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Should have results from task1 and task3 (CollectAll continued execution)
	if result.Get("task1") == nil {
		t.Error("Expected result from task1")
	}
	if result.Get("task3") == nil {
		t.Error("Expected result from task3")
	}

	// Should have one error
	if len(result.Errors()) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors()))
	}
}

// BenchmarkSequentialBuilder_Execute benchmarks sequential execution performance
func BenchmarkSequentialBuilder_Execute(b *testing.B) {
	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errors.FailFast}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create new sequential for each iteration since they can only execute once
		testSeq := Sequential(
			task.Task(func() (int, error) { return 1, nil }),
			task.Task(func() (int, error) { return 2, nil }),
			task.Task(func() (int, error) { return 3, nil }),
		)

		result, err := testSeq.Execute(ctx, cfg)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
		if result == nil {
			b.Fatal("Expected non-nil result")
		}
	}
}
