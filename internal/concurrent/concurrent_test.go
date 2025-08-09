package concurrent

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	internalErrors "github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/task"
	"github.com/maniartech/orchestrator/types"
)

// TestConcurrentBuilder_Basic tests basic concurrent execution functionality
func TestConcurrentBuilder_Basic(t *testing.T) {
	tests := []struct {
		name           string
		taskCount      int
		expectedValues []string
	}{
		{
			name:           "single task",
			taskCount:      1,
			expectedValues: []string{"task-0"},
		},
		{
			name:           "multiple tasks",
			taskCount:      3,
			expectedValues: []string{"task-0", "task-1", "task-2"},
		},
		{
			name:           "many tasks",
			taskCount:      10,
			expectedValues: []string{"task-0", "task-1", "task-2", "task-3", "task-4", "task-5", "task-6", "task-7", "task-8", "task-9"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create tasks
			var orchestrations []types.Orchestration
			for i := 0; i < tt.taskCount; i++ {
				value := fmt.Sprintf("task-%d", i)
				orchestrations = append(orchestrations, task.Task(func() (string, error) {
					return value, nil
				}))
			}

			// Create concurrent orchestration
			concurrent := Concurrent(orchestrations...)

			// Execute
			ctx := context.Background()
			result, err := concurrent.Execute(ctx, config.DefaultConfig())

			// Verify
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			// Check that all tasks completed
			for i, expectedValue := range tt.expectedValues {
				stepName := fmt.Sprintf("step-%d", i)
				actualValue := result.Get(stepName)
				if actualValue != expectedValue {
					t.Errorf("Expected result[%s] = %s, got: %v", stepName, expectedValue, actualValue)
				}
			}
		})
	}
}

// TestConcurrentBuilder_Named tests named concurrent orchestrations
func TestConcurrentBuilder_Named(t *testing.T) {
	task1 := task.Task(func() (string, error) { return "result1", nil }).Named("task1")
	task2 := task.Task(func() (string, error) { return "result2", nil }).Named("task2")

	concurrent := Concurrent(task1, task2).Named("test-concurrent")

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if concurrent.GetName() != "test-concurrent" {
		t.Errorf("Expected name 'test-concurrent', got: %s", concurrent.GetName())
	}

	// Check named results
	if result.Get("task1") != "result1" {
		t.Errorf("Expected result['task1'] = 'result1', got: %v", result.Get("task1"))
	}
	if result.Get("task2") != "result2" {
		t.Errorf("Expected result['task2'] = 'result2', got: %v", result.Get("task2"))
	}
}

// TestConcurrentBuilder_Configuration tests configuration inheritance
func TestConcurrentBuilder_Configuration(t *testing.T) {
	task1 := task.Task(func() (string, error) { return "result1", nil })
	task2 := task.Task(func() (string, error) { return "result2", nil })

	testConfig := config.Config{
		MaxConcurrency: 5,
		Timeout:        10 * time.Second,
		ErrorStrategy:  internalErrors.CollectAll,
	}

	concurrent := Concurrent(task1, task2).With(testConfig)

	if concurrent.GetConfig() == nil {
		t.Fatal("Expected config to be set")
	}

	if concurrent.GetConfig().MaxConcurrency != 5 {
		t.Errorf("Expected MaxConcurrency = 5, got: %d", concurrent.GetConfig().MaxConcurrency)
	}

	if concurrent.GetConfig().Timeout != 10*time.Second {
		t.Errorf("Expected Timeout = 10s, got: %v", concurrent.GetConfig().Timeout)
	}

	if concurrent.GetConfig().ErrorStrategy != internalErrors.CollectAll {
		t.Errorf("Expected ErrorStrategy = CollectAll, got: %v", concurrent.GetConfig().ErrorStrategy)
	}
}

// TestConcurrentBuilder_ErrorBoundary tests error boundary configuration
func TestConcurrentBuilder_ErrorBoundary(t *testing.T) {
	task1 := task.Task(func() (string, error) { return "result1", nil })
	task2 := task.Task(func() (string, error) { return "result2", nil })

	concurrent := Concurrent(task1, task2).ErrorBoundary(internalErrors.FailFast)

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}
}

// TestConcurrentBuilder_FailFast tests fail-fast error handling
func TestConcurrentBuilder_FailFast(t *testing.T) {
	var executionCount int32

	// Create tasks where the second one fails
	task1 := task.Task(func() (string, error) {
		atomic.AddInt32(&executionCount, 1)
		time.Sleep(100 * time.Millisecond) // Simulate work
		return "result1", nil
	})

	task2 := task.Task(func() (string, error) {
		atomic.AddInt32(&executionCount, 1)
		return "", errors.New("task2 failed")
	})

	task3 := task.Task(func() (string, error) {
		atomic.AddInt32(&executionCount, 1)
		time.Sleep(200 * time.Millisecond) // Simulate longer work
		return "result3", nil
	})

	concurrent := Concurrent(task1, task2, task3).ErrorBoundary(internalErrors.FailFast)

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	// Should have an error due to task2 failure
	if err == nil {
		t.Error("Expected error due to task2 failure")
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Should have errors in result
	if !result.HasErrors() {
		t.Error("Expected result to have errors")
	}

	// All tasks should have been started (due to concurrent execution)
	// but task3 might be cancelled early
	finalCount := atomic.LoadInt32(&executionCount)
	if finalCount < 2 {
		t.Errorf("Expected at least 2 tasks to start, got: %d", finalCount)
	}
}

// TestConcurrentBuilder_CollectAll tests collect-all error handling
func TestConcurrentBuilder_CollectAll(t *testing.T) {
	var executionCount int32

	// Create tasks where some fail
	task1 := task.Task(func() (string, error) {
		atomic.AddInt32(&executionCount, 1)
		return "result1", nil
	})

	task2 := task.Task(func() (string, error) {
		atomic.AddInt32(&executionCount, 1)
		return "", errors.New("task2 failed")
	})

	task3 := task.Task(func() (string, error) {
		atomic.AddInt32(&executionCount, 1)
		return "result3", nil
	})

	task4 := task.Task(func() (string, error) {
		atomic.AddInt32(&executionCount, 1)
		return "", errors.New("task4 failed")
	})

	concurrent := Concurrent(task1, task2, task3, task4).ErrorBoundary(internalErrors.CollectAll)

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	// Should have an error due to task failures
	if err == nil {
		t.Error("Expected error due to task failures")
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Should have errors in result
	if !result.HasErrors() {
		t.Error("Expected result to have errors")
	}

	// All tasks should have been executed
	finalCount := atomic.LoadInt32(&executionCount)
	if finalCount != 4 {
		t.Errorf("Expected all 4 tasks to execute, got: %d", finalCount)
	}

	// Should have successful results for task1 and task3
	if result.Get("step-0") != "result1" {
		t.Errorf("Expected result['step-0'] = 'result1', got: %v", result.Get("step-0"))
	}
	if result.Get("step-2") != "result3" {
		t.Errorf("Expected result['step-2'] = 'result3', got: %v", result.Get("step-2"))
	}

	// Should have 2 errors
	errors := result.Errors()
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got: %d", len(errors))
	}
}

// TestConcurrentBuilder_Timeout tests timeout handling
func TestConcurrentBuilder_Timeout(t *testing.T) {
	// Create a task that takes longer than the timeout
	slowTask := task.Task(func() (string, error) {
		time.Sleep(200 * time.Millisecond)
		return "slow result", nil
	})

	fastTask := task.Task(func() (string, error) {
		return "fast result", nil
	})

	concurrent := Concurrent(slowTask, fastTask).With(config.Config{
		Timeout: 100 * time.Millisecond,
	})

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	// Should have a timeout error
	if err == nil {
		t.Error("Expected timeout error")
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Should have errors due to timeout
	if !result.HasErrors() {
		t.Error("Expected result to have errors due to timeout")
	}
}

// TestConcurrentBuilder_Cancellation tests context cancellation
func TestConcurrentBuilder_Cancellation(t *testing.T) {
	var startedCount int32
	var completedCount int32

	// Create tasks that check for cancellation
	createTask := func(id int) types.Orchestration {
		return task.Task(func() (string, error) {
			atomic.AddInt32(&startedCount, 1)

			// Simulate work with cancellation checking
			for i := 0; i < 10; i++ {
				time.Sleep(20 * time.Millisecond)
			}

			atomic.AddInt32(&completedCount, 1)
			return fmt.Sprintf("result-%d", id), nil
		})
	}

	concurrent := Concurrent(
		createTask(1),
		createTask(2),
		createTask(3),
	)

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	// Should have a cancellation error
	if err == nil {
		t.Error("Expected cancellation error")
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Some tasks should have started
	started := atomic.LoadInt32(&startedCount)
	if started == 0 {
		t.Error("Expected some tasks to start")
	}

	// Not all tasks should have completed due to cancellation
	completed := atomic.LoadInt32(&completedCount)
	if completed >= started {
		t.Errorf("Expected fewer completions (%d) than starts (%d) due to cancellation", completed, started)
	}
}

// TestConcurrentBuilder_ConcurrencyLimit tests concurrency limiting
func TestConcurrentBuilder_ConcurrencyLimit(t *testing.T) {
	const maxConcurrency = 2
	const taskCount = 6

	var currentConcurrency int32
	var maxObservedConcurrency int32

	// Create tasks that track concurrency
	var orchestrations []types.Orchestration
	for i := 0; i < taskCount; i++ {
		value := fmt.Sprintf("task-%d", i)
		orchestrations = append(orchestrations, task.Task(func() (string, error) {
			current := atomic.AddInt32(&currentConcurrency, 1)

			// Track maximum observed concurrency
			for {
				max := atomic.LoadInt32(&maxObservedConcurrency)
				if current <= max || atomic.CompareAndSwapInt32(&maxObservedConcurrency, max, current) {
					break
				}
			}

			// Simulate work
			time.Sleep(100 * time.Millisecond)

			atomic.AddInt32(&currentConcurrency, -1)
			return value, nil
		}))
	}

	concurrent := Concurrent(orchestrations...).With(config.Config{
		MaxConcurrency: maxConcurrency,
	})

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check that concurrency was limited
	maxObserved := atomic.LoadInt32(&maxObservedConcurrency)
	if maxObserved > maxConcurrency {
		t.Errorf("Expected max concurrency <= %d, observed: %d", maxConcurrency, maxObserved)
	}

	// All tasks should have completed
	for i := 0; i < taskCount; i++ {
		stepName := fmt.Sprintf("step-%d", i)
		expectedValue := fmt.Sprintf("task-%d", i)
		if result.Get(stepName) != expectedValue {
			t.Errorf("Expected result[%s] = %s, got: %v", stepName, expectedValue, result.Get(stepName))
		}
	}
}

// TestConcurrentBuilder_ChildAccess tests child orchestration access methods
func TestConcurrentBuilder_ChildAccess(t *testing.T) {
	task1 := task.Task(func() (string, error) { return "result1", nil }).Named("task1")
	task2 := task.Task(func() (string, error) { return "result2", nil }).Named("task2")
	task3 := task.Task(func() (string, error) { return "result3", nil }) // unnamed

	concurrent := Concurrent(task1, task2, task3)

	// Test GetChildCount
	if concurrent.GetChildCount() != 3 {
		t.Errorf("Expected child count = 3, got: %d", concurrent.GetChildCount())
	}

	// Test GetChildAt
	child0, err := concurrent.GetChildAt(0)
	if err != nil {
		t.Errorf("Expected no error for GetChildAt(0), got: %v", err)
	}
	if child0.GetName() != "task1" {
		t.Errorf("Expected child0 name = 'task1', got: %s", child0.GetName())
	}

	// Test GetChildAt with invalid index
	_, err = concurrent.GetChildAt(5)
	if err == nil {
		t.Error("Expected error for GetChildAt(5)")
	}

	// Test GetChildren
	children := concurrent.GetChildren()
	if len(children) != 3 {
		t.Errorf("Expected 3 children, got: %d", len(children))
	}

	// Test FindChildByName
	index, child := concurrent.FindChildByName("task2")
	if index != 1 {
		t.Errorf("Expected index = 1 for 'task2', got: %d", index)
	}
	if child == nil || child.GetName() != "task2" {
		t.Errorf("Expected to find 'task2', got: %v", child)
	}

	// Test FindChildByName with generated name
	index, child = concurrent.FindChildByName("step-2")
	if index != 2 {
		t.Errorf("Expected index = 2 for 'step-2', got: %d", index)
	}
	if child == nil {
		t.Error("Expected to find 'step-2'")
	}

	// Test GetChildNames
	names := concurrent.GetChildNames()
	expectedNames := []string{"task1", "task2", "step-2"}
	if len(names) != len(expectedNames) {
		t.Errorf("Expected %d names, got: %d", len(expectedNames), len(names))
	}
	for i, expected := range expectedNames {
		if names[i] != expected {
			t.Errorf("Expected name[%d] = %s, got: %s", i, expected, names[i])
		}
	}
}

// TestConcurrentBuilder_Status tests status management
func TestConcurrentBuilder_Status(t *testing.T) {
	task1 := task.Task(func() (string, error) { return "result1", nil })
	concurrent := Concurrent(task1)

	// Initial status should be NotStarted
	if concurrent.GetStatus() != types.NotStarted {
		t.Errorf("Expected initial status = NotStarted, got: %v", concurrent.GetStatus())
	}

	// Execute
	ctx := context.Background()
	_, err := concurrent.Execute(ctx, config.DefaultConfig())

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Status should be Completed after successful execution
	if concurrent.GetStatus() != types.Completed {
		t.Errorf("Expected final status = Completed, got: %v", concurrent.GetStatus())
	}

	// Second execution should fail
	_, err = concurrent.Execute(ctx, config.DefaultConfig())
	if err == nil {
		t.Error("Expected error on second execution")
	}
}

// TestConcurrentBuilder_PanicRecovery tests panic recovery in concurrent tasks
func TestConcurrentBuilder_PanicRecovery(t *testing.T) {
	goodTask := task.Task(func() (string, error) { return "good result", nil })
	panicTask := task.Task(func() (string, error) {
		panic("test panic")
	})

	concurrent := Concurrent(goodTask, panicTask).ErrorBoundary(internalErrors.CollectAll)

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	// Should have an error due to panic
	if err == nil {
		t.Error("Expected error due to panic")
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Should have errors in result
	if !result.HasErrors() {
		t.Error("Expected result to have errors")
	}

	// Good task should still have completed
	if result.Get("step-0") != "good result" {
		t.Errorf("Expected result['step-0'] = 'good result', got: %v", result.Get("step-0"))
	}

	// Should have one error from panic
	errors := result.Errors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got: %d", len(errors))
	}

	// Error should mention panic
	if len(errors) > 0 && !contains(errors[0].Error.Error(), "panic") {
		t.Errorf("Expected error to mention panic, got: %v", errors[0].Error)
	}
}

// TestConcurrentBuilder_HighLoad tests high concurrency scenarios
func TestConcurrentBuilder_HighLoad(t *testing.T) {
	const taskCount = 100
	const maxConcurrency = 10

	var executionCount int32

	// Create many tasks
	var orchestrations []types.Orchestration
	for i := 0; i < taskCount; i++ {
		value := fmt.Sprintf("task-%d", i)
		orchestrations = append(orchestrations, task.Task(func() (string, error) {
			atomic.AddInt32(&executionCount, 1)
			// Small delay to simulate work
			time.Sleep(1 * time.Millisecond)
			return value, nil
		}))
	}

	concurrent := Concurrent(orchestrations...).With(config.Config{
		MaxConcurrency: maxConcurrency,
	})

	ctx := context.Background()
	start := time.Now()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// All tasks should have executed
	finalCount := atomic.LoadInt32(&executionCount)
	if finalCount != taskCount {
		t.Errorf("Expected %d executions, got: %d", taskCount, finalCount)
	}

	// Should complete in reasonable time (much faster than sequential)
	maxExpectedDuration := time.Duration(taskCount/maxConcurrency+1) * 10 * time.Millisecond
	if duration > maxExpectedDuration {
		t.Errorf("Expected duration <= %v, got: %v", maxExpectedDuration, duration)
	}

	// All results should be present
	for i := 0; i < taskCount; i++ {
		stepName := fmt.Sprintf("step-%d", i)
		expectedValue := fmt.Sprintf("task-%d", i)
		if result.Get(stepName) != expectedValue {
			t.Errorf("Expected result[%s] = %s, got: %v", stepName, expectedValue, result.Get(stepName))
		}
	}
}

// TestConcurrentBuilder_RaceConditions tests for race conditions
func TestConcurrentBuilder_RaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	const iterations = 100
	const tasksPerIteration = 10

	for i := 0; i < iterations; i++ {
		var orchestrations []types.Orchestration
		for j := 0; j < tasksPerIteration; j++ {
			value := fmt.Sprintf("task-%d-%d", i, j)
			orchestrations = append(orchestrations, task.Task(func() (string, error) {
				// Random small delay to increase chance of race conditions
				time.Sleep(time.Duration(j%5) * time.Millisecond)
				return value, nil
			}))
		}

		concurrent := Concurrent(orchestrations...)

		ctx := context.Background()
		result, err := concurrent.Execute(ctx, config.DefaultConfig())

		if err != nil {
			t.Errorf("Iteration %d: Expected no error, got: %v", i, err)
		}

		if result == nil {
			t.Fatalf("Iteration %d: Expected result, got nil", i)
		}

		// Verify all results are present
		for j := 0; j < tasksPerIteration; j++ {
			stepName := fmt.Sprintf("step-%d", j)
			expectedValue := fmt.Sprintf("task-%d-%d", i, j)
			if result.Get(stepName) != expectedValue {
				t.Errorf("Iteration %d: Expected result[%s] = %s, got: %v", i, stepName, expectedValue, result.Get(stepName))
			}
		}
	}
}

// TestConcurrentBuilder_GoroutineLeaks tests for goroutine leaks
func TestConcurrentBuilder_GoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping goroutine leak test in short mode")
	}

	initialGoroutines := runtime.NumGoroutine()

	// Run multiple concurrent executions
	for i := 0; i < 10; i++ {
		var orchestrations []types.Orchestration
		for j := 0; j < 5; j++ {
			orchestrations = append(orchestrations, task.Task(func() (string, error) {
				time.Sleep(10 * time.Millisecond)
				return "result", nil
			}))
		}

		concurrent := Concurrent(orchestrations...)
		ctx := context.Background()
		_, err := concurrent.Execute(ctx, config.DefaultConfig())

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	}

	// Allow some time for goroutines to clean up
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()

	// Allow for some variance in goroutine count, but should not grow significantly
	if finalGoroutines > initialGoroutines+5 {
		t.Errorf("Potential goroutine leak: started with %d, ended with %d goroutines",
			initialGoroutines, finalGoroutines)
	}
}

// TestConcurrentBuilder_ValidationErrors tests validation error cases
func TestConcurrentBuilder_ValidationErrors(t *testing.T) {
	// Test panic on empty orchestrations
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for empty orchestrations")
		}
	}()

	Concurrent()
}

// TestConcurrentBuilder_NilOrchestration tests validation for nil orchestrations
func TestConcurrentBuilder_NilOrchestration(t *testing.T) {
	task1 := task.Task(func() (string, error) { return "result1", nil })

	// Test panic on nil orchestration
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil orchestration")
		}
	}()

	Concurrent(task1, nil)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsAt(s, substr))))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
