package concurrent

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	internalErrors "github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/types"

	. "github.com/maniartech/orchestrator/pkg/builders/concurrent"
)

// TestConcurrentSynchronization tests WaitGroup-based synchronization
func TestConcurrentSynchronization(t *testing.T) {
	const taskCount = 20
	var completedCount int32
	var startedCount int32

	// Create tasks that track execution
	var orchestrations []types.Orchestration
	for i := 0; i < taskCount; i++ {
		orchestrations = append(orchestrations, task.Task(func() (string, error) {
			atomic.AddInt32(&startedCount, 1)
			time.Sleep(10 * time.Millisecond) // Simulate work
			atomic.AddInt32(&completedCount, 1)
			return "completed", nil
		}))
	}

	concurrent := Concurrent(orchestrations...)

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// All tasks should have started and completed
	finalStarted := atomic.LoadInt32(&startedCount)
	finalCompleted := atomic.LoadInt32(&completedCount)

	if finalStarted != taskCount {
		t.Errorf("Expected %d tasks to start, got: %d", taskCount, finalStarted)
	}

	if finalCompleted != taskCount {
		t.Errorf("Expected %d tasks to complete, got: %d", taskCount, finalCompleted)
	}
}

// TestAtomicErrorCollection tests atomic error collection
func TestAtomicErrorCollection(t *testing.T) {
	const taskCount = 50
	const errorRate = 0.3 // 30% of tasks will fail

	var errorCount int32
	var successCount int32

	// Create tasks with controlled failure rate
	var orchestrations []types.Orchestration
	for i := 0; i < taskCount; i++ {
		shouldFail := float64(i)/float64(taskCount) < errorRate

		if shouldFail {
			orchestrations = append(orchestrations, task.Task(func() (string, error) {
				atomic.AddInt32(&errorCount, 1)
				return "", errors.New("task failed")
			}))
		} else {
			orchestrations = append(orchestrations, task.Task(func() (string, error) {
				atomic.AddInt32(&successCount, 1)
				return "success", nil
			}))
		}
	}

	concurrent := Concurrent(orchestrations...).ErrorBoundary(internalErrors.CollectAll)

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	// Should have an error due to failures
	if err == nil {
		t.Error("Expected error due to task failures")
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check error collection
	if !result.HasErrors() {
		t.Error("Expected result to have errors")
	}

	resultErrors := result.Errors()
	expectedErrorCount := int(atomic.LoadInt32(&errorCount))

	if len(resultErrors) != expectedErrorCount {
		t.Errorf("Expected %d errors in result, got: %d", expectedErrorCount, len(resultErrors))
	}

	// All tasks should have executed (collect-all strategy)
	finalErrorCount := atomic.LoadInt32(&errorCount)
	finalSuccessCount := atomic.LoadInt32(&successCount)
	totalExecuted := finalErrorCount + finalSuccessCount

	if int(totalExecuted) != taskCount {
		t.Errorf("Expected %d total executions, got: %d", taskCount, totalExecuted)
	}
}

// TestFailFastCancellation tests fail-fast cancellation behavior
func TestFailFastCancellation(t *testing.T) {
	var startedCount int32
	var completedCount int32

	// Create tasks where one fails quickly and others take longer
	fastFailTask := task.Task(func() (string, error) {
		atomic.AddInt32(&startedCount, 1)
		return "", errors.New("fast failure")
	})

	var slowTasks []types.Orchestration
	for i := 0; i < 10; i++ {
		slowTasks = append(slowTasks, task.Task(func() (string, error) {
			atomic.AddInt32(&startedCount, 1)

			// Check for cancellation during execution
			for j := 0; j < 10; j++ {
				time.Sleep(20 * time.Millisecond)
			}

			atomic.AddInt32(&completedCount, 1)
			return "completed", nil
		}))
	}

	// Mix fast-failing task with slow tasks
	var orchestrations []types.Orchestration
	orchestrations = append(orchestrations, fastFailTask)
	orchestrations = append(orchestrations, slowTasks...)

	concurrent := Concurrent(orchestrations...).ErrorBoundary(internalErrors.FailFast)

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	// Should have an error due to fast failure
	if err == nil {
		t.Error("Expected error due to fast failure")
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Some tasks should have started
	finalStarted := atomic.LoadInt32(&startedCount)
	if finalStarted == 0 {
		t.Error("Expected some tasks to start")
	}

	// Not all tasks should have completed due to cancellation
	finalCompleted := atomic.LoadInt32(&completedCount)
	if finalCompleted >= finalStarted {
		t.Errorf("Expected fewer completions (%d) than starts (%d) due to cancellation",
			finalCompleted, finalStarted)
	}

	// Should have at least one error
	if !result.HasErrors() {
		t.Error("Expected result to have errors")
	}
}

// TestConcurrencyLimitSynchronization tests semaphore-based concurrency control
func TestConcurrencyLimitSynchronization(t *testing.T) {
	const taskCount = 20
	const maxConcurrency = 3

	var currentConcurrency int32
	var maxObservedConcurrency int32
	var executionOrder []int
	var mu sync.Mutex

	// Create tasks that track concurrency
	var orchestrations []types.Orchestration
	for i := 0; i < taskCount; i++ {
		taskID := i
		orchestrations = append(orchestrations, task.Task(func() (string, error) {
			// Track execution order
			mu.Lock()
			executionOrder = append(executionOrder, taskID)
			mu.Unlock()

			// Track current concurrency
			current := atomic.AddInt32(&currentConcurrency, 1)

			// Update max observed concurrency
			for {
				max := atomic.LoadInt32(&maxObservedConcurrency)
				if current <= max || atomic.CompareAndSwapInt32(&maxObservedConcurrency, max, current) {
					break
				}
			}

			// Simulate work
			time.Sleep(50 * time.Millisecond)

			atomic.AddInt32(&currentConcurrency, -1)
			return "completed", nil
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

	// Check that concurrency was properly limited
	maxObserved := atomic.LoadInt32(&maxObservedConcurrency)
	if maxObserved > int32(maxConcurrency) {
		t.Errorf("Expected max concurrency <= %d, observed: %d", maxConcurrency, maxObserved)
	}

	// All tasks should have executed
	mu.Lock()
	executedCount := len(executionOrder)
	mu.Unlock()

	if executedCount != taskCount {
		t.Errorf("Expected %d tasks to execute, got: %d", taskCount, executedCount)
	}

	// Final concurrency should be zero
	finalConcurrency := atomic.LoadInt32(&currentConcurrency)
	if finalConcurrency != 0 {
		t.Errorf("Expected final concurrency to be 0, got: %d", finalConcurrency)
	}
}

// TestResourceCleanup tests proper resource cleanup
func TestResourceCleanup(t *testing.T) {
	const taskCount = 10

	var resourcesAcquired int32
	var resourcesReleased int32

	// Create tasks that simulate resource acquisition/release
	var orchestrations []types.Orchestration
	for i := 0; i < taskCount; i++ {
		orchestrations = append(orchestrations, task.Task(func() (string, error) {
			// Simulate resource acquisition
			atomic.AddInt32(&resourcesAcquired, 1)

			// Simulate work
			time.Sleep(10 * time.Millisecond)

			// Simulate resource release (this should happen even if task fails)
			defer atomic.AddInt32(&resourcesReleased, 1)

			return "completed", nil
		}))
	}

	concurrent := Concurrent(orchestrations...)

	ctx := context.Background()
	result, err := concurrent.Execute(ctx, config.DefaultConfig())

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// All resources should be properly cleaned up
	finalAcquired := atomic.LoadInt32(&resourcesAcquired)
	finalReleased := atomic.LoadInt32(&resourcesReleased)

	if finalAcquired != int32(taskCount) {
		t.Errorf("Expected %d resources acquired, got: %d", taskCount, finalAcquired)
	}

	if finalReleased != int32(taskCount) {
		t.Errorf("Expected %d resources released, got: %d", taskCount, finalReleased)
	}

	if finalAcquired != finalReleased {
		t.Errorf("Resource leak detected: acquired %d, released %d", finalAcquired, finalReleased)
	}
}

// TestContextCancellationPropagation tests context cancellation propagation
func TestContextCancellationPropagation(t *testing.T) {
	const taskCount = 10

	var startedCount int32

	// Create tasks that can detect cancellation
	var orchestrations []types.Orchestration
	for i := 0; i < taskCount; i++ {
		orchestrations = append(orchestrations, task.Task(func() (string, error) {
			atomic.AddInt32(&startedCount, 1)

			// Simulate work with cancellation checking
			for j := 0; j < 10; j++ {
				time.Sleep(20 * time.Millisecond)
			}

			return "completed", nil
		}))
	}

	concurrentOrch := Concurrent(orchestrations...)

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result, err := concurrentOrch.Execute(ctx, config.DefaultConfig())

	// Should have a cancellation error
	if err == nil {
		t.Error("Expected cancellation error")
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Some tasks should have started
	finalStarted := atomic.LoadInt32(&startedCount)
	if finalStarted == 0 {
		t.Error("Expected some tasks to start")
	}

	// Should have errors due to cancellation
	if !result.HasErrors() {
		t.Error("Expected result to have errors due to cancellation")
	}
}

// TestZeroAllocationErrorCollection tests that error collection doesn't allocate unnecessarily
func TestZeroAllocationErrorCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping allocation test in short mode")
	}

	const taskCount = 5

	// Create tasks that succeed (no errors to collect)
	var orchestrations []types.Orchestration
	for i := 0; i < taskCount; i++ {
		orchestrations = append(orchestrations, task.Task(func() (string, error) {
			return "success", nil
		}))
	}

	ctx := context.Background()
	config := config.DefaultConfig()

	// Measure allocations
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Execute multiple times to get stable measurements
	for i := 0; i < 10; i++ {
		// Create fresh orchestrations for each iteration
		var freshOrchestrations []types.Orchestration
		for j := 0; j < taskCount; j++ {
			freshOrchestrations = append(freshOrchestrations, task.Task(func() (string, error) {
				return "success", nil
			}))
		}

		concurrent := Concurrent(freshOrchestrations...)
		_, err := concurrent.Execute(ctx, config)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check that allocations are reasonable (not zero due to goroutine overhead, but should be bounded)
	allocsPerOp := (m2.Mallocs - m1.Mallocs) / 10
	if allocsPerOp > 200 { // Allow reasonable allocations for goroutines, channels, and sync primitives
		t.Errorf("Too many allocations per operation: %d", allocsPerOp)
	}

	// Log the allocation count for reference
	t.Logf("Allocations per operation: %d", allocsPerOp)
}
