package orchestrator_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/maniartech/orchestrator"
	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/result"
	"github.com/maniartech/orchestrator/internal/status"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/types"
)

// TestRaceCondition_TaskExecution tests concurrent task execution for race conditions
func TestRaceCondition_TaskExecution(t *testing.T) {
	const numGoroutines = 100
	const numIterations = 10

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	ctx := context.Background()
	cfg := config.Config{
		Timeout: 5 * time.Second,
	}

	for i := 0; i < numIterations; i++ {
		wg.Add(numGoroutines)

		for j := 0; j < numGoroutines; j++ {
			go func(iteration, goroutineID int) {
				defer wg.Done()

				taskFn := func() (string, error) {
					// Simulate some work
					time.Sleep(time.Microsecond)
					return fmt.Sprintf("result-%d-%d", iteration, goroutineID), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("race-task-%d-%d", iteration, goroutineID))
				result, err := taskInstance.Execute(ctx, cfg)

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("Task execution failed: %v", err)
				} else {
					atomic.AddInt64(&successCount, 1)
					if result == nil {
						t.Errorf("Result is nil for task %d-%d", iteration, goroutineID)
					}
				}
			}(i, j)
		}

		wg.Wait()
	}

	expectedTotal := int64(numGoroutines * numIterations)
	actualTotal := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&errorCount)

	if actualTotal != expectedTotal {
		t.Errorf("Expected %d total operations, got %d", expectedTotal, actualTotal)
	}

	t.Logf("Race condition test completed: %d successes, %d errors out of %d operations",
		atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount), expectedTotal)
}

// TestRaceCondition_WorkflowExecution tests concurrent workflow execution
func TestRaceCondition_WorkflowExecution(t *testing.T) {
	const numGoroutines = 50
	const numIterations = 5

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	for i := 0; i < numIterations; i++ {
		wg.Add(numGoroutines)

		for j := 0; j < numGoroutines; j++ {
			go func(iteration, goroutineID int) {
				defer wg.Done()

				taskFn := func() (string, error) {
					return fmt.Sprintf("workflow-result-%d-%d", iteration, goroutineID), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("workflow-race-task-%d-%d", iteration, goroutineID))
				workflow := Setup(taskInstance)

				result, err := workflow.Await()

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("Workflow execution failed: %v", err)
				} else {
					atomic.AddInt64(&successCount, 1)
					if result == nil {
						t.Errorf("Result is nil for workflow %d-%d", iteration, goroutineID)
					}
				}
			}(i, j)
		}

		wg.Wait()
	}

	expectedTotal := int64(numGoroutines * numIterations)
	actualTotal := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&errorCount)

	if actualTotal != expectedTotal {
		t.Errorf("Expected %d total operations, got %d", expectedTotal, actualTotal)
	}

	t.Logf("Workflow race condition test completed: %d successes, %d errors out of %d operations",
		atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount), expectedTotal)
}

// TestRaceCondition_StatusManagement tests concurrent status operations
func TestRaceCondition_StatusManagement(t *testing.T) {
	const numGoroutines = 100
	const numOperations = 1000

	manager := status.NewManager()
	var wg sync.WaitGroup
	var operationCount int64

	// Test concurrent status operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Perform various status operations
				switch j % 4 {
				case 0:
					manager.Set(status.Status(j % 5))
				case 1:
					_ = manager.Get()
				case 2:
					manager.CompareAndSwap(status.Status(0), status.Status(1))
				case 3:
					manager.CompareAndSwap(status.Status(1), status.Status(0))
				}

				atomic.AddInt64(&operationCount, 1)
			}
		}(i)
	}

	wg.Wait()

	expectedOperations := int64(numGoroutines * numOperations)
	actualOperations := atomic.LoadInt64(&operationCount)

	if actualOperations != expectedOperations {
		t.Errorf("Expected %d operations, got %d", expectedOperations, actualOperations)
	}

	t.Logf("Status management race condition test completed: %d operations", actualOperations)
}

// TestRaceCondition_ResultOperations tests concurrent result operations
func TestRaceCondition_ResultOperations(t *testing.T) {
	const numGoroutines = 50
	const numOperations = 100

	r := result.NewResult()
	var wg sync.WaitGroup
	var setCount int64
	var getCount int64

	// Test concurrent result operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", goroutineID, j)
				value := fmt.Sprintf("value-%d-%d", goroutineID, j)

				// Set operation
				r.Set(key, value)
				atomic.AddInt64(&setCount, 1)

				// Get operation
				retrieved := r.Get(key)
				atomic.AddInt64(&getCount, 1)

				if retrieved != value {
					t.Errorf("Expected value %s, got %v for key %s", value, retrieved, key)
				}
			}
		}(i)
	}

	wg.Wait()

	expectedOps := int64(numGoroutines * numOperations)

	if atomic.LoadInt64(&setCount) != expectedOps {
		t.Errorf("Expected %d set operations, got %d", expectedOps, atomic.LoadInt64(&setCount))
	}

	if atomic.LoadInt64(&getCount) != expectedOps {
		t.Errorf("Expected %d get operations, got %d", expectedOps, atomic.LoadInt64(&getCount))
	}

	t.Logf("Result operations race condition test completed: %d sets, %d gets",
		atomic.LoadInt64(&setCount), atomic.LoadInt64(&getCount))
}

// TestRaceCondition_TaskStatusTransitions tests concurrent status transitions
func TestRaceCondition_TaskStatusTransitions(t *testing.T) {
	const numTasks = 100

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	ctx := context.Background()
	cfg := config.Config{}

	wg.Add(numTasks)
	for i := 0; i < numTasks; i++ {
		go func(taskID int) {
			defer wg.Done()

			taskFn := func() (string, error) {
				// Simulate work with random duration
				time.Sleep(time.Duration(taskID%10) * time.Microsecond)
				return fmt.Sprintf("task-%d-result", taskID), nil
			}

			taskInstance := task.Task(taskFn).Named(fmt.Sprintf("status-race-task-%d", taskID))

			// Check initial status
			if taskStatus := taskInstance.GetStatus(); taskStatus != types.NotStarted {
				t.Errorf("Expected NotStarted status, got %v", taskStatus)
			}

			// Execute task
			_, err := taskInstance.Execute(ctx, cfg)

			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}

			// Check final status
			finalStatus := taskInstance.GetStatus()
			if finalStatus != types.Completed && finalStatus != types.Failed {
				t.Errorf("Expected terminal status, got %v", finalStatus)
			}
		}(i)
	}

	wg.Wait()

	total := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&errorCount)
	if total != numTasks {
		t.Errorf("Expected %d total tasks, got %d", numTasks, total)
	}

	t.Logf("Task status transitions race test completed: %d successes, %d errors",
		atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount))
}

// TestRaceCondition_MemoryStability tests memory stability under concurrent load
func TestRaceCondition_MemoryStability(t *testing.T) {
	const numGoroutines = 20
	const numIterations = 50

	// Measure initial memory
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	var wg sync.WaitGroup
	var operationCount int64

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				taskFn := func() ([]string, error) {
					// Allocate some memory
					data := make([]string, 10)
					for k := range data {
						data[k] = fmt.Sprintf("item-%d-%d-%d", goroutineID, j, k)
					}
					return data, nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("memory-race-task-%d-%d", goroutineID, j))
				workflow := Setup(taskInstance)

				_, err := workflow.Await()
				if err != nil {
					t.Errorf("Memory stability test failed: %v", err)
				}

				atomic.AddInt64(&operationCount, 1)

				// Periodic GC
				if j%10 == 0 {
					runtime.GC()
				}
			}
		}(i)
	}

	wg.Wait()

	// Final memory measurement
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	memoryGrowth := int64(m2.Alloc - m1.Alloc)
	maxAcceptableGrowth := int64(10 * 1024 * 1024) // 10MB

	if memoryGrowth > maxAcceptableGrowth {
		t.Errorf("Memory growth too high: %d bytes (max acceptable: %d bytes)",
			memoryGrowth, maxAcceptableGrowth)
	}

	t.Logf("Memory stability test completed: %d operations, %d bytes memory growth",
		atomic.LoadInt64(&operationCount), memoryGrowth)
}

// TestRaceCondition_GoroutineLeaks tests for goroutine leaks under concurrent load
func TestRaceCondition_GoroutineLeaks(t *testing.T) {
	const numIterations = 50
	const tasksPerIteration = 10

	// Measure initial goroutines
	initialGoroutines := runtime.NumGoroutine()

	var wg sync.WaitGroup

	for i := 0; i < numIterations; i++ {
		wg.Add(tasksPerIteration)

		for j := 0; j < tasksPerIteration; j++ {
			go func(iteration, taskID int) {
				defer wg.Done()

				taskFn := func() (string, error) {
					time.Sleep(time.Microsecond)
					return fmt.Sprintf("leak-test-%d-%d", iteration, taskID), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("leak-test-task-%d-%d", iteration, taskID))
				workflow := Setup(taskInstance)

				_, err := workflow.Await()
				if err != nil {
					t.Errorf("Goroutine leak test failed: %v", err)
				}
			}(i, j)
		}

		wg.Wait()

		// Periodic check for goroutine growth
		if i%10 == 0 {
			runtime.GC()
			time.Sleep(10 * time.Millisecond) // Allow goroutines to clean up
		}
	}

	// Final cleanup and measurement
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	finalGoroutines := runtime.NumGoroutine()

	goroutineGrowth := finalGoroutines - initialGoroutines
	maxAcceptableGrowth := 10 // Allow some growth for test infrastructure

	if goroutineGrowth > maxAcceptableGrowth {
		t.Errorf("Goroutine leak detected: %d goroutines growth (max acceptable: %d)",
			goroutineGrowth, maxAcceptableGrowth)
	}

	t.Logf("Goroutine leak test completed: %d goroutines growth (initial: %d, final: %d)",
		goroutineGrowth, initialGoroutines, finalGoroutines)
}

// TestRaceCondition_ConcurrentCancellation tests concurrent cancellation scenarios
func TestRaceCondition_ConcurrentCancellation(t *testing.T) {
	const numGoroutines = 20
	const numTasks = 5

	var wg sync.WaitGroup
	var cancelledCount int64
	var completedCount int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)

		go func(goroutineID int) {
			defer wg.Done()

			ctx, cancel := context.WithCancel(context.Background())

			// Start multiple tasks
			var taskWg sync.WaitGroup
			taskWg.Add(numTasks)

			for j := 0; j < numTasks; j++ {
				go func(taskID int) {
					defer taskWg.Done()

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
						return fmt.Sprintf("completed-%d-%d", goroutineID, taskID), nil
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("cancel-race-task-%d-%d", goroutineID, taskID))
					workflow := Setup(taskInstance)

					result, err := workflow.AwaitWithContext(ctx)

					if err != nil && err == context.Canceled {
						atomic.AddInt64(&cancelledCount, 1)
					} else if err == nil && result != nil {
						atomic.AddInt64(&completedCount, 1)
					}
				}(j)
			}

			// Cancel after a random delay
			go func() {
				time.Sleep(time.Duration(goroutineID%10) * time.Microsecond)
				cancel()
			}()

			taskWg.Wait()
		}(i)
	}

	wg.Wait()

	totalOperations := atomic.LoadInt64(&cancelledCount) + atomic.LoadInt64(&completedCount)
	expectedTotal := int64(numGoroutines * numTasks)

	if totalOperations != expectedTotal {
		t.Errorf("Expected %d total operations, got %d", expectedTotal, totalOperations)
	}

	t.Logf("Concurrent cancellation test completed: %d cancelled, %d completed out of %d",
		atomic.LoadInt64(&cancelledCount), atomic.LoadInt64(&completedCount), expectedTotal)
}

// TestRaceCondition_StressTest runs a comprehensive stress test for race conditions
func TestRaceCondition_StressTest(t *testing.T) {
	const duration = 2 * time.Second
	const numGoroutines = 10

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup
	var operationCount int64
	var errorCount int64

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			operationID := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					taskFn := func() (string, error) {
						// Random work simulation
						workType := operationID % 4
						switch workType {
						case 0:
							time.Sleep(time.Microsecond)
						case 1:
							// CPU intensive
							sum := 0
							for k := 0; k < 1000; k++ {
								sum += k
							}
						case 2:
							// Memory allocation
							data := make([]byte, 1024)
							for k := range data {
								data[k] = byte(k % 256)
							}
						case 3:
							// Error case
							if operationID%10 == 0 {
								return "", fmt.Errorf("stress test error %d", operationID)
							}
						}

						return fmt.Sprintf("stress-result-%d-%d", goroutineID, operationID), nil
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("stress-task-%d-%d", goroutineID, operationID))
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

	wg.Wait()

	totalOps := atomic.LoadInt64(&operationCount)
	totalErrors := atomic.LoadInt64(&errorCount)

	if totalOps == 0 {
		t.Error("No operations completed during stress test")
	}

	t.Logf("Stress test completed: %d operations, %d errors in %v",
		totalOps, totalErrors, duration)
}

// TestRaceCondition_AtomicOperations tests atomic operations under concurrent load
func TestRaceCondition_AtomicOperations(t *testing.T) {
	const numGoroutines = 50
	const numOperations = 1000

	var counter int64
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				atomic.AddInt64(&counter, 1)
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * numOperations)
	if counter != expected {
		t.Errorf("Expected counter %d, got %d", expected, counter)
	}

	t.Logf("Atomic operations test completed: %d operations", counter)
}

// TestRaceCondition_ChannelOperations tests channel operations for race conditions
func TestRaceCondition_ChannelOperations(t *testing.T) {
	const numGoroutines = 20
	const numMessages = 100

	ch := make(chan string, numMessages*numGoroutines)
	var wg sync.WaitGroup

	// Producers
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				ch <- fmt.Sprintf("message-%d-%d", id, j)
			}
		}(i)
	}

	// Consumer
	var receivedCount int64
	var consumerWG sync.WaitGroup
	consumerWG.Add(1)
	go func() {
		defer consumerWG.Done()
		for range ch {
			atomic.AddInt64(&receivedCount, 1)
		}
	}()

	wg.Wait()
	close(ch)

	// Wait for consumer to drain the channel completely
	consumerWG.Wait()

	expected := int64(numGoroutines * numMessages)
	actual := atomic.LoadInt64(&receivedCount)
	if actual != expected {
		t.Errorf("Expected %d messages, received %d", expected, actual)
	}

	t.Logf("Channel operations test completed: %d messages", actual)
}

// TestRaceCondition_MapOperations tests concurrent map operations
func TestRaceCondition_MapOperations(t *testing.T) {
	const numGoroutines = 20
	const numOperations = 100

	m := make(map[string]int)
	var mu sync.RWMutex
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)

				// Write operation
				mu.Lock()
				m[key] = j
				mu.Unlock()

				// Read operation
				mu.RLock()
				_ = m[key]
				mu.RUnlock()
			}
		}(i)
	}

	wg.Wait()

	mu.RLock()
	mapSize := len(m)
	mu.RUnlock()

	expected := numGoroutines * numOperations
	if mapSize != expected {
		t.Errorf("Expected map size %d, got %d", expected, mapSize)
	}

	t.Logf("Map operations test completed: %d entries", mapSize)
}
