package orchestrator

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/task"
)

// TestProductionStress_HighThroughput tests high-throughput scenarios
func TestProductionStress_HighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production stress test in short mode")
	}

	const (
		numWorkers     = 20
		tasksPerWorker = 500
		totalTasks     = numWorkers * tasksPerWorker
		maxDuration    = 30 * time.Second
	)

	ctx, cancel := context.WithTimeout(context.Background(), maxDuration)
	defer cancel()

	var wg sync.WaitGroup
	var completedTasks int64
	var errorTasks int64
	var totalDuration int64 // in nanoseconds

	start := time.Now()

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < tasksPerWorker; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					taskStart := time.Now()

					taskFn := func() (string, error) {
						// Simulate realistic work
						workType := j % 5
						switch workType {
						case 0: // Fast computation
							sum := 0
							for k := 0; k < 100; k++ {
								sum += k
							}
						case 1: // I/O simulation
							time.Sleep(100 * time.Microsecond)
						case 2: // Memory allocation
							data := make([]byte, 1024)
							for k := range data {
								data[k] = byte(k % 256)
							}
						case 3: // String processing
							result := fmt.Sprintf("worker-%d-task-%d", workerID, j)
							_ = len(result)
						case 4: // Error case (5% of tasks)
							if j%20 == 0 {
								return "", fmt.Errorf("simulated error in worker %d task %d", workerID, j)
							}
						}

						return fmt.Sprintf("result-%d-%d", workerID, j), nil
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("stress-task-%d-%d", workerID, j))
					workflow := Setup(taskInstance)

					_, err := workflow.Await()

					taskDuration := time.Since(taskStart)
					atomic.AddInt64(&totalDuration, taskDuration.Nanoseconds())

					if err != nil {
						atomic.AddInt64(&errorTasks, 1)
					} else {
						atomic.AddInt64(&completedTasks, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(start)

	completed := atomic.LoadInt64(&completedTasks)
	errors := atomic.LoadInt64(&errorTasks)
	avgDuration := time.Duration(atomic.LoadInt64(&totalDuration) / (completed + errors))

	throughput := float64(completed+errors) / totalTime.Seconds()

	t.Logf("High throughput stress test completed:")
	t.Logf("  Total time: %v", totalTime)
	t.Logf("  Completed tasks: %d", completed)
	t.Logf("  Error tasks: %d", errors)
	t.Logf("  Total processed: %d/%d", completed+errors, totalTasks)
	t.Logf("  Throughput: %.2f tasks/second", throughput)
	t.Logf("  Average task duration: %v", avgDuration)

	// Verify minimum performance requirements
	minThroughput := 1000.0 // 1000 tasks/second minimum
	if throughput < minThroughput {
		t.Errorf("Throughput too low: %.2f tasks/second (minimum: %.2f)", throughput, minThroughput)
	}

	// Verify completion rate
	completionRate := float64(completed+errors) / float64(totalTasks)
	if completionRate < 0.95 { // 95% completion rate minimum
		t.Errorf("Completion rate too low: %.2f%% (minimum: 95%%)", completionRate*100)
	}
}

// TestProductionStress_MemoryStability tests memory stability under sustained load
func TestProductionStress_MemoryStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production stress test in short mode")
	}

	const (
		testDuration   = 10 * time.Second
		numWorkers     = 10
		allocationSize = 10 * 1024 // 10KB per task
		gcInterval     = 1 * time.Second
	)

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// Measure initial memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	var wg sync.WaitGroup
	var operationCount int64
	var allocationCount int64

	// Start GC monitoring
	gcTicker := time.NewTicker(gcInterval)
	defer gcTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-gcTicker.C:
				runtime.GC()
			}
		}
	}()

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
					taskFn := func() ([]byte, error) {
						// Allocate memory
						data := make([]byte, allocationSize)
						for j := range data {
							data[j] = byte((workerID + operationID + j) % 256)
						}

						atomic.AddInt64(&allocationCount, 1)

						// Simulate processing
						time.Sleep(time.Microsecond)

						return data, nil
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("memory-task-%d-%d", workerID, operationID))
					workflow := Setup(taskInstance)

					_, err := workflow.Await()
					if err != nil {
						t.Errorf("Memory task failed: %v", err)
						return
					}

					atomic.AddInt64(&operationCount, 1)
					operationID++
				}
			}
		}(i)
	}

	wg.Wait()

	// Final memory measurement
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	operations := atomic.LoadInt64(&operationCount)
	allocations := atomic.LoadInt64(&allocationCount)
	memoryGrowth := int64(m2.Alloc - m1.Alloc)
	totalAllocated := int64(m2.TotalAlloc - m1.TotalAlloc)

	t.Logf("Memory stability stress test completed:")
	t.Logf("  Duration: %v", testDuration)
	t.Logf("  Operations: %d", operations)
	t.Logf("  Allocations: %d", allocations)
	t.Logf("  Memory growth: %d bytes (%.2f MB)", memoryGrowth, float64(memoryGrowth)/(1024*1024))
	t.Logf("  Total allocated: %d bytes (%.2f MB)", totalAllocated, float64(totalAllocated)/(1024*1024))
	t.Logf("  Peak memory: %d bytes (%.2f MB)", m2.Sys, float64(m2.Sys)/(1024*1024))
	t.Logf("  GC cycles: %d", m2.NumGC-m1.NumGC)

	// Check for memory leaks
	maxAcceptableGrowth := int64(50 * 1024 * 1024) // 50MB
	if memoryGrowth > maxAcceptableGrowth {
		t.Errorf("Memory growth too high: %d bytes (max acceptable: %d bytes)", memoryGrowth, maxAcceptableGrowth)
	}

	// Verify operations completed
	if operations == 0 {
		t.Error("No operations completed during memory stability test")
	}
}

// TestProductionStress_GoroutineLifecycle tests goroutine lifecycle management
func TestProductionStress_GoroutineLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production stress test in short mode")
	}

	const (
		numCycles     = 100
		tasksPerCycle = 20
		cycleInterval = 50 * time.Millisecond
	)

	// Measure initial goroutines
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	var totalTasks int64
	var completedTasks int64
	maxGoroutines := initialGoroutines

	for cycle := 0; cycle < numCycles; cycle++ {
		var wg sync.WaitGroup
		wg.Add(tasksPerCycle)

		for i := 0; i < tasksPerCycle; i++ {
			atomic.AddInt64(&totalTasks, 1)

			go func(cycleID, taskID int) {
				defer wg.Done()
				defer func() {
					atomic.AddInt64(&completedTasks, 1)
				}()

				taskFn := func() (string, error) {
					// Simulate work with goroutine creation
					var innerWg sync.WaitGroup
					innerWg.Add(3)

					results := make([]string, 3)
					for j := 0; j < 3; j++ {
						go func(idx int) {
							defer innerWg.Done()
							time.Sleep(time.Microsecond)
							results[idx] = fmt.Sprintf("inner-%d", idx)
						}(j)
					}

					innerWg.Wait()
					return fmt.Sprintf("cycle-%d-task-%d", cycleID, taskID), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("lifecycle-task-%d-%d", cycleID, taskID))
				workflow := Setup(taskInstance)

				_, err := workflow.Await()
				if err != nil {
					t.Errorf("Lifecycle task failed: %v", err)
				}
			}(cycle, i)
		}

		wg.Wait()

		// Monitor goroutine count
		currentGoroutines := runtime.NumGoroutine()
		if currentGoroutines > maxGoroutines {
			maxGoroutines = currentGoroutines
		}

		// Periodic cleanup and check
		if cycle%20 == 0 {
			runtime.GC()
			time.Sleep(10 * time.Millisecond)

			goroutineGrowth := currentGoroutines - initialGoroutines
			t.Logf("Cycle %d: %d goroutines (growth: %d)", cycle, currentGoroutines, goroutineGrowth)

			// Check for excessive growth during execution
			if goroutineGrowth > 100 {
				t.Errorf("Excessive goroutine growth at cycle %d: %d", cycle, goroutineGrowth)
			}
		}

		time.Sleep(cycleInterval)
	}

	// Final cleanup and measurement
	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	finalGoroutines := runtime.NumGoroutine()

	totalGrowth := finalGoroutines - initialGoroutines
	peakGrowth := maxGoroutines - initialGoroutines

	t.Logf("Goroutine lifecycle test completed:")
	t.Logf("  Total tasks: %d", atomic.LoadInt64(&totalTasks))
	t.Logf("  Completed tasks: %d", atomic.LoadInt64(&completedTasks))
	t.Logf("  Initial goroutines: %d", initialGoroutines)
	t.Logf("  Final goroutines: %d", finalGoroutines)
	t.Logf("  Peak goroutines: %d", maxGoroutines)
	t.Logf("  Final growth: %d", totalGrowth)
	t.Logf("  Peak growth: %d", peakGrowth)

	// Verify task completion
	if atomic.LoadInt64(&completedTasks) != atomic.LoadInt64(&totalTasks) {
		t.Errorf("Not all tasks completed: %d/%d", atomic.LoadInt64(&completedTasks), atomic.LoadInt64(&totalTasks))
	}

	// Check for goroutine leaks
	maxAcceptableGrowth := 30
	if totalGrowth > maxAcceptableGrowth {
		t.Errorf("Goroutine leak detected: %d goroutines growth (max acceptable: %d)", totalGrowth, maxAcceptableGrowth)
	}
}

// TestProductionStress_ErrorResilience tests error handling under stress
func TestProductionStress_ErrorResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production stress test in short mode")
	}

	const (
		testDuration = 5 * time.Second
		numWorkers   = 15
		errorRate    = 0.3 // 30% error rate
	)

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	var wg sync.WaitGroup
	var totalOperations int64
	var successOperations int64
	var errorOperations int64
	var panicOperations int64
	var timeoutOperations int64

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
						// Simulate various error conditions
						errorType := float64(operationID%100) / 100.0

						if errorType < errorRate/3 {
							// Regular error
							return "", fmt.Errorf("simulated error %d-%d", workerID, operationID)
						} else if errorType < 2*errorRate/3 {
							// Panic
							panic(fmt.Sprintf("simulated panic %d-%d", workerID, operationID))
						} else if errorType < errorRate {
							// Timeout (long operation)
							time.Sleep(200 * time.Millisecond)
							return fmt.Sprintf("timeout-result-%d-%d", workerID, operationID), nil
						} else {
							// Success
							return fmt.Sprintf("success-result-%d-%d", workerID, operationID), nil
						}
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("resilience-task-%d-%d", workerID, operationID))
					workflow := Setup(taskInstance)

					// Random timeout
					timeout := 100 * time.Millisecond
					taskCtx, taskCancel := context.WithTimeout(context.Background(), timeout)

					result, err := workflow.AwaitWithContext(taskCtx)
					taskCancel()

					atomic.AddInt64(&totalOperations, 1)

					if err != nil {
						if err == context.DeadlineExceeded {
							atomic.AddInt64(&timeoutOperations, 1)
						} else {
							atomic.AddInt64(&errorOperations, 1)
						}
					} else if result != nil && result.HasErrors() {
						// Check for panic recovery
						errors := result.Errors()
						hasPanic := false
						for _, opErr := range errors {
							if opErr.Stack != nil {
								hasPanic = true
								break
							}
						}
						if hasPanic {
							atomic.AddInt64(&panicOperations, 1)
						} else {
							atomic.AddInt64(&errorOperations, 1)
						}
					} else {
						atomic.AddInt64(&successOperations, 1)
					}

					operationID++
				}
			}
		}(i)
	}

	wg.Wait()

	total := atomic.LoadInt64(&totalOperations)
	success := atomic.LoadInt64(&successOperations)
	errors := atomic.LoadInt64(&errorOperations)
	panics := atomic.LoadInt64(&panicOperations)
	timeouts := atomic.LoadInt64(&timeoutOperations)

	successRate := float64(success) / float64(total) * 100
	calculatedErrorRate := float64(errors) / float64(total) * 100
	panicRate := float64(panics) / float64(total) * 100
	timeoutRate := float64(timeouts) / float64(total) * 100

	t.Logf("Error resilience stress test completed:")
	t.Logf("  Duration: %v", testDuration)
	t.Logf("  Total operations: %d", total)
	t.Logf("  Success: %d (%.2f%%)", success, successRate)
	t.Logf("  Errors: %d (%.2f%%)", errors, calculatedErrorRate)
	t.Logf("  Panics recovered: %d (%.2f%%)", panics, panicRate)
	t.Logf("  Timeouts: %d (%.2f%%)", timeouts, timeoutRate)

	// Verify system remained stable
	if total == 0 {
		t.Error("No operations completed during error resilience test")
	}

	// Verify panic recovery worked
	if panics > 0 {
		t.Logf("Successfully recovered from %d panics", panics)
	}

	// Verify the system handled errors gracefully
	if success+errors+panics+timeouts != total {
		t.Errorf("Operation count mismatch: %d + %d + %d + %d != %d", success, errors, panics, timeouts, total)
	}
}

// TestProductionStress_ConcurrentCancellation tests cancellation under concurrent load
func TestProductionStress_ConcurrentCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production stress test in short mode")
	}

	const (
		numIterations     = 50
		tasksPerIteration = 30
		cancellationDelay = 20 * time.Millisecond
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
							time.Sleep(100 * time.Microsecond)
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

	cancellationRate := float64(atomic.LoadInt64(&totalCancelled)) / float64(totalOperations) * 100
	completionRate := float64(atomic.LoadInt64(&totalCompleted)) / float64(totalOperations) * 100

	t.Logf("Concurrent cancellation stress test completed:")
	t.Logf("  Total operations: %d/%d", totalOperations, expectedTotal)
	t.Logf("  Cancelled: %d (%.2f%%)", atomic.LoadInt64(&totalCancelled), cancellationRate)
	t.Logf("  Completed: %d (%.2f%%)", atomic.LoadInt64(&totalCompleted), completionRate)
	t.Logf("  Errors: %d", atomic.LoadInt64(&totalErrors))

	if totalOperations != expectedTotal {
		t.Errorf("Expected %d total operations, got %d", expectedTotal, totalOperations)
	}

	// Should have some cancellations
	if atomic.LoadInt64(&totalCancelled) == 0 {
		t.Error("Expected some cancellations, got none")
	}

	// Verify cancellation worked effectively
	if cancellationRate < 50 { // At least 50% should be cancelled
		t.Errorf("Cancellation rate too low: %.2f%% (expected at least 50%%)", cancellationRate)
	}
}
