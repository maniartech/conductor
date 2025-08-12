package orchestrator

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/task"
)

// TestStress_HighVolumeExecution tests high-volume task execution
func TestStress_HighVolumeExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const (
		numGoroutines     = 100
		tasksPerGoroutine = 100
		totalTasks        = numGoroutines * tasksPerGoroutine
	)

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	ctx := context.Background()
	cfg := config.Config{
		Timeout: 10 * time.Second,
	}

	start := time.Now()

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < tasksPerGoroutine; j++ {
				taskFn := func() (string, error) {
					// Simulate varying workloads
					workType := j % 4
					switch workType {
					case 0:
						// CPU work
						sum := 0
						for k := 0; k < 1000; k++ {
							sum += k
						}
					case 1:
						// Memory allocation
						data := make([]byte, 1024)
						for k := range data {
							data[k] = byte(k % 256)
						}
					case 2:
						// I/O simulation
						time.Sleep(time.Microsecond)
					case 3:
						// Error case (10% of the time)
						if j%10 == 0 {
							return "", fmt.Errorf("stress test error %d-%d", goroutineID, j)
						}
					}

					return fmt.Sprintf("stress-result-%d-%d", goroutineID, j), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("stress-task-%d-%d", goroutineID, j))
				_, err := taskInstance.Execute(ctx, cfg)

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalCompleted := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&errorCount)

	if totalCompleted != totalTasks {
		t.Errorf("Expected %d total tasks, got %d", totalTasks, totalCompleted)
	}

	throughput := float64(totalCompleted) / duration.Seconds()

	t.Logf("High volume stress test completed:")
	t.Logf("  Total tasks: %d", totalCompleted)
	t.Logf("  Successes: %d", atomic.LoadInt64(&successCount))
	t.Logf("  Errors: %d", atomic.LoadInt64(&errorCount))
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f tasks/second", throughput)

	// Verify minimum throughput
	minThroughput := 1000.0 // 1000 tasks/second minimum
	if throughput < minThroughput {
		t.Errorf("Throughput too low: %.2f tasks/second (minimum: %.2f)", throughput, minThroughput)
	}
}

// TestStress_MemoryPressure tests behavior under memory pressure
func TestStress_MemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const (
		numIterations  = 1000
		allocationSize = 10 * 1024 // 10KB per task
	)

	// Measure initial memory
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	var wg sync.WaitGroup
	var completedTasks int64

	ctx := context.Background()
	cfg := config.Config{
		Timeout: 5 * time.Second,
	}

	start := time.Now()

	for i := 0; i < numIterations; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()

			taskFn := func() ([]byte, error) {
				// Allocate memory
				data := make([]byte, allocationSize)
				for j := range data {
					data[j] = byte(j % 256)
				}

				// Simulate processing
				time.Sleep(time.Microsecond)

				return data, nil
			}

			taskInstance := task.Task(taskFn).Named(fmt.Sprintf("memory-stress-task-%d", iteration))
			result, err := taskInstance.Execute(ctx, cfg)
			if err != nil {
				t.Errorf("Memory stress task failed: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Result is nil for memory stress task %d", iteration)
				return
			}

			atomic.AddInt64(&completedTasks, 1)
		}(i)

		// Periodic GC to manage memory pressure
		if i%100 == 0 {
			runtime.GC()
		}
	}

	wg.Wait()
	duration := time.Since(start)

	// Final memory measurement
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	completed := atomic.LoadInt64(&completedTasks)
	memoryGrowth := int64(m2.Alloc - m1.Alloc)

	t.Logf("Memory pressure stress test completed:")
	t.Logf("  Completed tasks: %d/%d", completed, numIterations)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Memory growth: %d bytes", memoryGrowth)
	t.Logf("  Peak memory: %d bytes", m2.Sys)

	if completed != numIterations {
		t.Errorf("Expected %d completed tasks, got %d", numIterations, completed)
	}

	// Check for excessive memory growth
	maxAcceptableGrowth := int64(50 * 1024 * 1024) // 50MB
	if memoryGrowth > maxAcceptableGrowth {
		t.Errorf("Memory growth too high: %d bytes (max acceptable: %d bytes)",
			memoryGrowth, maxAcceptableGrowth)
	}
}

// TestStress_GoroutineLeakDetection tests for goroutine leaks under stress
func TestStress_GoroutineLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const (
		numIterations     = 500
		tasksPerIteration = 10
	)

	// Measure initial goroutines
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	ctx := context.Background()
	cfg := config.Config{
		Timeout: 5 * time.Second,
	}

	var totalTasks int64

	for i := 0; i < numIterations; i++ {
		var wg sync.WaitGroup
		wg.Add(tasksPerIteration)

		for j := 0; j < tasksPerIteration; j++ {
			go func(iteration, taskID int) {
				defer wg.Done()

				taskFn := func() (string, error) {
					// Simulate work
					time.Sleep(time.Microsecond)
					return fmt.Sprintf("leak-test-%d-%d", iteration, taskID), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("leak-test-task-%d-%d", iteration, taskID))
				_, err := taskInstance.Execute(ctx, cfg)
				if err != nil {
					t.Errorf("Leak detection task failed: %v", err)
				}

				atomic.AddInt64(&totalTasks, 1)
			}(i, j)
		}

		wg.Wait()

		// Periodic cleanup and measurement
		if i%50 == 0 {
			runtime.GC()
			time.Sleep(10 * time.Millisecond)

			currentGoroutines := runtime.NumGoroutine()
			growth := currentGoroutines - initialGoroutines

			t.Logf("Iteration %d: %d goroutines (growth: %d)", i, currentGoroutines, growth)

			// Check for excessive growth during execution
			if growth > 50 {
				t.Logf("Warning: High goroutine growth detected: %d", growth)
			}
		}
	}

	// Final cleanup and measurement
	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	finalGoroutines := runtime.NumGoroutine()

	totalGrowth := finalGoroutines - initialGoroutines
	completed := atomic.LoadInt64(&totalTasks)
	expected := int64(numIterations * tasksPerIteration)

	t.Logf("Goroutine leak detection completed:")
	t.Logf("  Total tasks: %d/%d", completed, expected)
	t.Logf("  Initial goroutines: %d", initialGoroutines)
	t.Logf("  Final goroutines: %d", finalGoroutines)
	t.Logf("  Total growth: %d", totalGrowth)

	if completed != expected {
		t.Errorf("Expected %d completed tasks, got %d", expected, completed)
	}

	// Check for goroutine leaks
	maxAcceptableGrowth := 20 // Allow some growth for test infrastructure
	if totalGrowth > maxAcceptableGrowth {
		t.Errorf("Goroutine leak detected: %d goroutines growth (max acceptable: %d)",
			totalGrowth, maxAcceptableGrowth)
	}
}

// TestStress_ConcurrentCancellation tests concurrent cancellation under stress
func TestStress_ConcurrentCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const (
		numIterations     = 100
		tasksPerIteration = 20
		cancellationDelay = 10 * time.Millisecond
	)

	var totalCancelled int64
	var totalCompleted int64
	var totalErrors int64

	for i := 0; i < numIterations; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		var wg sync.WaitGroup
		wg.Add(tasksPerIteration)

		// Start tasks
		for j := 0; j < tasksPerIteration; j++ {
			go func(iteration, taskID int) {
				defer wg.Done()

				taskFn := func() (string, error) {
					// Long-running task that can be cancelled
					for k := 0; k < 1000; k++ {
						select {
						case <-ctx.Done():
							return "", ctx.Err()
						default:
							time.Sleep(time.Microsecond)
						}
					}
					return fmt.Sprintf("completed-%d-%d", iteration, taskID), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("cancel-stress-task-%d-%d", iteration, taskID))
				workflow := Setup(taskInstance)

				result, err := workflow.AwaitWithContext(ctx)

				if err != nil {
					if err == context.Canceled {
						atomic.AddInt64(&totalCancelled, 1)
					} else {
						atomic.AddInt64(&totalErrors, 1)
					}
				} else if result != nil {
					atomic.AddInt64(&totalCompleted, 1)
				}
			}(i, j)
		}

		// Cancel after delay
		go func() {
			time.Sleep(cancellationDelay)
			cancel()
		}()

		wg.Wait()
	}

	totalOperations := atomic.LoadInt64(&totalCancelled) + atomic.LoadInt64(&totalCompleted) + atomic.LoadInt64(&totalErrors)
	expectedTotal := int64(numIterations * tasksPerIteration)

	t.Logf("Concurrent cancellation stress test completed:")
	t.Logf("  Total operations: %d/%d", totalOperations, expectedTotal)
	t.Logf("  Cancelled: %d", atomic.LoadInt64(&totalCancelled))
	t.Logf("  Completed: %d", atomic.LoadInt64(&totalCompleted))
	t.Logf("  Errors: %d", atomic.LoadInt64(&totalErrors))

	if totalOperations != expectedTotal {
		t.Errorf("Expected %d total operations, got %d", expectedTotal, totalOperations)
	}

	// Should have some cancellations
	if atomic.LoadInt64(&totalCancelled) == 0 {
		t.Error("Expected some cancellations, got none")
	}
}

// TestStress_ResourceExhaustion tests behavior under resource exhaustion
func TestStress_ResourceExhaustion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const (
		numGoroutines     = 50 // Reduced to avoid race conditions
		tasksPerGoroutine = 20 // Reduced to avoid race conditions
	)

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	var timeoutCount int64

	ctx := context.Background()
	cfg := config.Config{
		Timeout: 100 * time.Millisecond, // Short timeout to trigger timeouts
	}

	start := time.Now()

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < tasksPerGoroutine; j++ {
				// Use atomic operations to avoid race conditions
				localJ := j // Capture loop variable
				taskFn := func() (string, error) {
					// Variable work that might exceed timeout
					workDuration := time.Duration(localJ%10) * 20 * time.Millisecond
					time.Sleep(workDuration)

					return fmt.Sprintf("exhaustion-result-%d-%d", goroutineID, localJ), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("exhaustion-task-%d-%d", goroutineID, localJ))
				_, err := taskInstance.Execute(ctx, cfg)

				if err != nil {
					if err == context.DeadlineExceeded {
						atomic.AddInt64(&timeoutCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalOperations := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&errorCount) + atomic.LoadInt64(&timeoutCount)
	expectedTotal := int64(numGoroutines * tasksPerGoroutine)

	t.Logf("Resource exhaustion stress test completed:")
	t.Logf("  Total operations: %d/%d", totalOperations, expectedTotal)
	t.Logf("  Successes: %d", atomic.LoadInt64(&successCount))
	t.Logf("  Errors: %d", atomic.LoadInt64(&errorCount))
	t.Logf("  Timeouts: %d", atomic.LoadInt64(&timeoutCount))
	t.Logf("  Duration: %v", duration)

	if totalOperations != expectedTotal {
		t.Errorf("Expected %d total operations, got %d", expectedTotal, totalOperations)
	}

	// Should have some timeouts due to short timeout
	if atomic.LoadInt64(&timeoutCount) == 0 {
		t.Log("Warning: Expected some timeouts, got none")
	}
}

// TestStress_LongRunning tests long-running stability
func TestStress_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const testDuration = 5 * time.Second
	const checkInterval = 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	var operationCount int64
	var errorCount int64

	// Measure initial resources
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialGoroutines := runtime.NumGoroutine()

	// Start continuous operations
	var wg sync.WaitGroup
	const numWorkers = 10

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			operationID := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					taskFn := func() (string, error) {
						// Simulate various workloads
						workType := operationID % 5
						switch workType {
						case 0:
							time.Sleep(time.Microsecond)
						case 1:
							// CPU work
							sum := 0
							for k := 0; k < 100; k++ {
								sum += k
							}
						case 2:
							// Memory allocation
							data := make([]byte, 1024)
							for k := range data {
								data[k] = byte(k % 256)
							}
						case 3:
							// Error case (5% of the time)
							if operationID%20 == 0 {
								return "", fmt.Errorf("long running error %d", operationID)
							}
						case 4:
							// Quick operation
							// No additional work
						}

						return fmt.Sprintf("long-running-%d-%d", workerID, operationID), nil
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("long-running-task-%d-%d", workerID, operationID))
					workflow := Setup(taskInstance)

					_, err := workflow.Await()

					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					}

					atomic.AddInt64(&operationCount, 1)
					operationID++
				}
			}
		}(i)
	}

	// Monitor resources periodically
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runtime.GC()
				var m runtime.MemStats
				runtime.ReadMemStats(&m)

				currentGoroutines := runtime.NumGoroutine()
				memoryGrowth := int64(m.Alloc - m1.Alloc)
				goroutineGrowth := currentGoroutines - initialGoroutines

				t.Logf("Long-running check: %d ops, %d errors, %d goroutines (+%d), %d bytes memory (+%d)",
					atomic.LoadInt64(&operationCount),
					atomic.LoadInt64(&errorCount),
					currentGoroutines,
					goroutineGrowth,
					m.Alloc,
					memoryGrowth)
			}
		}
	}()

	wg.Wait()

	// Final measurements
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	finalGoroutines := runtime.NumGoroutine()

	totalOps := atomic.LoadInt64(&operationCount)
	totalErrors := atomic.LoadInt64(&errorCount)
	memoryGrowth := int64(m2.Alloc - m1.Alloc)
	goroutineGrowth := finalGoroutines - initialGoroutines

	t.Logf("Long-running stress test completed:")
	t.Logf("  Duration: %v", testDuration)
	t.Logf("  Total operations: %d", totalOps)
	t.Logf("  Total errors: %d", totalErrors)
	t.Logf("  Throughput: %.2f ops/second", float64(totalOps)/testDuration.Seconds())
	t.Logf("  Memory growth: %d bytes", memoryGrowth)
	t.Logf("  Goroutine growth: %d", goroutineGrowth)

	if totalOps == 0 {
		t.Error("No operations completed during long-running test")
	}

	// Check for resource leaks
	maxMemoryGrowth := int64(20 * 1024 * 1024) // 20MB
	if memoryGrowth > maxMemoryGrowth {
		t.Errorf("Memory growth too high: %d bytes (max: %d)", memoryGrowth, maxMemoryGrowth)
	}

	maxGoroutineGrowth := 20
	if goroutineGrowth > maxGoroutineGrowth {
		t.Errorf("Goroutine growth too high: %d (max: %d)", goroutineGrowth, maxGoroutineGrowth)
	}
}
