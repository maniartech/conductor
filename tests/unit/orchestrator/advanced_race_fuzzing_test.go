package orchestrator_test

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	. "github.com/maniartech/orchestrator"
	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/task"
)

// TestAdvancedRaceConditions_ConcurrentWorkflowExecution tests multiple workflows executing concurrently
func TestAdvancedRaceConditions_ConcurrentWorkflowExecution(t *testing.T) {
	const numWorkflows = 100
	const tasksPerWorkflow = 5

	var wg sync.WaitGroup
	var completedWorkflows int64
	var totalTasks int64

	wg.Add(numWorkflows)
	for i := 0; i < numWorkflows; i++ {
		go func(workflowID int) {
			defer wg.Done()

			var taskWg sync.WaitGroup
			taskWg.Add(tasksPerWorkflow)

			for j := 0; j < tasksPerWorkflow; j++ {
				go func(taskID int) {
					defer taskWg.Done()

					taskFn := func() (string, error) {
						// Simulate work with random duration
						time.Sleep(time.Duration(rand.Intn(10)) * time.Microsecond)
						return fmt.Sprintf("workflow-%d-task-%d", workflowID, taskID), nil
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("workflow-%d-task-%d", workflowID, taskID))
					workflow := Setup(taskInstance)

					_, err := workflow.Await()
					if err != nil {
						t.Errorf("Task failed: %v", err)
					}

					atomic.AddInt64(&totalTasks, 1)
				}(j)
			}

			taskWg.Wait()
			atomic.AddInt64(&completedWorkflows, 1)
		}(i)
	}

	wg.Wait()

	expectedWorkflows := int64(numWorkflows)
	expectedTasks := int64(numWorkflows * tasksPerWorkflow)

	if completedWorkflows != expectedWorkflows {
		t.Errorf("Expected %d workflows, completed %d", expectedWorkflows, completedWorkflows)
	}

	if totalTasks != expectedTasks {
		t.Errorf("Expected %d tasks, completed %d", expectedTasks, totalTasks)
	}

	t.Logf("Concurrent workflow execution test completed: %d workflows, %d tasks", completedWorkflows, totalTasks)
}

// TestAdvancedRaceConditions_StatusTransitions tests concurrent status transitions
func TestAdvancedRaceConditions_StatusTransitions(t *testing.T) {
	const numGoroutines = 50
	const numOperations = 1000

	var wg sync.WaitGroup
	var transitionCount int64

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				taskFn := func() (string, error) {
					return fmt.Sprintf("status-test-%d-%d", goroutineID, j), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("status-task-%d-%d", goroutineID, j))

				// Check initial status
				if status := taskInstance.GetStatus(); status.String() != "NotStarted" {
					t.Errorf("Expected NotStarted, got %s", status.String())
				}

				// Execute and check final status
				ctx := context.Background()
				cfg := config.Config{}
				_, err := taskInstance.Execute(ctx, cfg)

				if err != nil {
					if status := taskInstance.GetStatus(); status.String() != "Failed" {
						t.Errorf("Expected Failed status after error, got %s", status.String())
					}
				} else {
					if status := taskInstance.GetStatus(); status.String() != "Completed" {
						t.Errorf("Expected Completed status after success, got %s", status.String())
					}
				}

				atomic.AddInt64(&transitionCount, 1)
			}
		}(i)
	}

	wg.Wait()

	expectedTransitions := int64(numGoroutines * numOperations)
	if transitionCount != expectedTransitions {
		t.Errorf("Expected %d transitions, got %d", expectedTransitions, transitionCount)
	}

	t.Logf("Status transitions test completed: %d transitions", transitionCount)
}

// TestAdvancedRaceConditions_MemoryBarriers tests memory barriers and visibility
func TestAdvancedRaceConditions_MemoryBarriers(t *testing.T) {
	const numGoroutines = 20
	const numIterations = 1000

	var sharedData int64
	var readCount int64
	var writeCount int64
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				if j%2 == 0 {
					// Writer
					atomic.StoreInt64(&sharedData, int64(goroutineID*1000+j))
					atomic.AddInt64(&writeCount, 1)
				} else {
					// Reader
					value := atomic.LoadInt64(&sharedData)
					if value < 0 {
						t.Errorf("Invalid shared data value: %d", value)
					}
					atomic.AddInt64(&readCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	expectedOperations := int64(numGoroutines * numIterations)
	totalOperations := atomic.LoadInt64(&readCount) + atomic.LoadInt64(&writeCount)

	if totalOperations != expectedOperations {
		t.Errorf("Expected %d operations, got %d", expectedOperations, totalOperations)
	}

	t.Logf("Memory barriers test completed: %d reads, %d writes", readCount, writeCount)
}

// TestAdvancedRaceConditions_PointerOperations tests concurrent pointer operations
func TestAdvancedRaceConditions_PointerOperations(t *testing.T) {
	const numGoroutines = 30
	const numOperations = 500

	type Data struct {
		Value int64
		ID    string
	}

	var sharedPtr unsafe.Pointer
	var operationCount int64
	var wg sync.WaitGroup

	// Initialize with initial data
	initialData := &Data{Value: 0, ID: "initial"}
	atomic.StorePointer(&sharedPtr, unsafe.Pointer(initialData))

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				if j%3 == 0 {
					// Writer - create new data
					newData := &Data{
						Value: int64(goroutineID*1000 + j),
						ID:    fmt.Sprintf("goroutine-%d-op-%d", goroutineID, j),
					}
					atomic.StorePointer(&sharedPtr, unsafe.Pointer(newData))
				} else {
					// Reader - read current data
					ptr := atomic.LoadPointer(&sharedPtr)
					if ptr != nil {
						data := (*Data)(ptr)
						if data.Value < 0 {
							t.Errorf("Invalid data value: %d", data.Value)
						}
					}
				}

				atomic.AddInt64(&operationCount, 1)
			}
		}(i)
	}

	wg.Wait()

	expectedOperations := int64(numGoroutines * numOperations)
	if operationCount != expectedOperations {
		t.Errorf("Expected %d operations, got %d", expectedOperations, operationCount)
	}

	// Verify final state
	finalPtr := atomic.LoadPointer(&sharedPtr)
	if finalPtr == nil {
		t.Error("Final pointer should not be nil")
	}

	t.Logf("Pointer operations test completed: %d operations", operationCount)
}

// TestAdvancedRaceConditions_GoroutineLeakDetection tests for goroutine leaks with detailed monitoring
func TestAdvancedRaceConditions_GoroutineLeakDetection(t *testing.T) {
	const numIterations = 100
	const tasksPerIteration = 10

	// Measure initial goroutines
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	var totalTasks int64
	var completedTasks int64

	for i := 0; i < numIterations; i++ {
		var wg sync.WaitGroup
		wg.Add(tasksPerIteration)

		for j := 0; j < tasksPerIteration; j++ {
			atomic.AddInt64(&totalTasks, 1)

			go func(iteration, taskID int) {
				defer wg.Done()
				defer func() {
					atomic.AddInt64(&completedTasks, 1)
				}()

				taskFn := func() (string, error) {
					// Simulate work
					time.Sleep(time.Microsecond)
					return fmt.Sprintf("leak-test-%d-%d", iteration, taskID), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("leak-test-task-%d-%d", iteration, taskID))
				ctx := context.Background()
				cfg := config.Config{}

				_, err := taskInstance.Execute(ctx, cfg)
				if err != nil {
					t.Errorf("Task execution failed: %v", err)
				}
			}(i, j)
		}

		wg.Wait()

		// Periodic cleanup and check
		if i%20 == 0 {
			runtime.GC()
			time.Sleep(10 * time.Millisecond)

			currentGoroutines := runtime.NumGoroutine()
			growth := currentGoroutines - initialGoroutines

			if growth > 50 {
				t.Logf("Warning: High goroutine growth at iteration %d: %d goroutines", i, growth)
			}
		}
	}

	// Final cleanup and measurement
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	finalGoroutines := runtime.NumGoroutine()

	totalGrowth := finalGoroutines - initialGoroutines
	expectedTasks := int64(numIterations * tasksPerIteration)

	if totalTasks != expectedTasks {
		t.Errorf("Expected %d total tasks, got %d", expectedTasks, totalTasks)
	}

	if completedTasks != expectedTasks {
		t.Errorf("Expected %d completed tasks, got %d", expectedTasks, completedTasks)
	}

	// Check for goroutine leaks
	maxAcceptableGrowth := 20
	if totalGrowth > maxAcceptableGrowth {
		t.Errorf("Goroutine leak detected: %d goroutines growth (max acceptable: %d)", totalGrowth, maxAcceptableGrowth)
	}

	t.Logf("Goroutine leak detection completed: %d tasks, %d goroutines growth", completedTasks, totalGrowth)
}

// TestAdvancedRaceConditions_ChaosEngineering tests system resilience under chaotic conditions
func TestAdvancedRaceConditions_ChaosEngineering(t *testing.T) {
	const duration = 3 * time.Second
	const numChaosGoroutines = 5

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var operationCount int64
	var errorCount int64
	var panicCount int64
	var timeoutCount int64
	var wg sync.WaitGroup

	wg.Add(numChaosGoroutines)
	for i := 0; i < numChaosGoroutines; i++ {
		go func(chaosID int) {
			defer wg.Done()

			operationID := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Chaos engineering - random behaviors
					ct := rand.Intn(10) // snapshot chaos type

					// snapshot operation id to avoid data race with concurrent goroutines
					opID := operationID

					taskFn := func() (string, error) {
						switch ct {
						case 0, 1, 2, 3: // Normal operation (40%)
							time.Sleep(time.Duration(rand.Intn(1000)) * time.Microsecond)
							return fmt.Sprintf("chaos-normal-%d-%d", chaosID, opID), nil

						case 4, 5: // Error cases (20%)
							return "", fmt.Errorf("chaos error %d-%d", chaosID, opID)

						case 6: // Panic case (10%)
							panic(fmt.Sprintf("chaos panic %d-%d", chaosID, opID))

						case 7: // Long running task (10%)
							time.Sleep(100 * time.Millisecond)
							return fmt.Sprintf("chaos-long-%d-%d", chaosID, opID), nil

						case 8: // Memory intensive (10%)
							data := make([]byte, 10*1024) // 10KB
							for k := range data {
								data[k] = byte(k % 256)
							}
							return fmt.Sprintf("chaos-memory-%d-%d", chaosID, opID), nil

						case 9: // CPU intensive (10%)
							sum := 0
							for k := 0; k < 10000; k++ {
								sum += k
							}
							return fmt.Sprintf("chaos-cpu-%d-%d-%d", chaosID, opID, sum), nil

						default:
							return fmt.Sprintf("chaos-default-%d-%d", chaosID, opID), nil
						}
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("chaos-task-%d-%d", chaosID, opID))

					// Random timeout
					timeout := time.Duration(rand.Intn(50)+10) * time.Millisecond
					cfg := config.Config{Timeout: timeout}

					taskCtx := context.Background()
					result, err := taskInstance.Execute(taskCtx, cfg)

					if err != nil {
						if err == context.DeadlineExceeded {
							atomic.AddInt64(&timeoutCount, 1)
						} else {
							atomic.AddInt64(&errorCount, 1)
						}
					} else if result != nil {
						// Check for panic recovery in results
						if result.HasErrors() {
							errors := result.Errors()
							for _, opErr := range errors {
								if opErr.Stack != nil {
									atomic.AddInt64(&panicCount, 1)
									break
								}
							}
						}
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
	totalPanics := atomic.LoadInt64(&panicCount)
	totalTimeouts := atomic.LoadInt64(&timeoutCount)

	if totalOps == 0 {
		t.Error("No operations completed during chaos test")
	}

	t.Logf("Chaos engineering test completed in %v:", duration)
	t.Logf("  Total operations: %d", totalOps)
	t.Logf("  Errors: %d", totalErrors)
	t.Logf("  Panics recovered: %d", totalPanics)
	t.Logf("  Timeouts: %d", totalTimeouts)
	t.Logf("  Success rate: %.2f%%", float64(totalOps-totalErrors-totalPanics-totalTimeouts)/float64(totalOps)*100)
}

// TestAdvancedRaceConditions_PropertyBasedTesting tests complex orchestration scenarios
func TestAdvancedRaceConditions_PropertyBasedTesting(t *testing.T) {
	const numProperties = 50
	const maxDepth = 5

	var wg sync.WaitGroup
	var propertyTests int64
	var propertyFailures int64

	wg.Add(numProperties)
	for i := 0; i < numProperties; i++ {
		go func(propertyID int) {
			defer wg.Done()

			// Generate random orchestration structure
			_ = rand.Intn(maxDepth) + 1 // depth for future use
			numTasks := rand.Intn(10) + 1

			var taskWg sync.WaitGroup
			taskWg.Add(numTasks)

			for j := 0; j < numTasks; j++ {
				go func(taskID int) {
					defer taskWg.Done()

					// Property: All tasks should complete successfully or fail gracefully
					taskFn := func() (interface{}, error) {
						// Random behavior
						behavior := rand.Intn(4)
						switch behavior {
						case 0: // Success
							return fmt.Sprintf("property-success-%d-%d", propertyID, taskID), nil
						case 1: // Error
							return nil, fmt.Errorf("property-error-%d-%d", propertyID, taskID)
						case 2: // Panic
							panic(fmt.Sprintf("property-panic-%d-%d", propertyID, taskID))
						case 3: // Slow task
							time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
							return fmt.Sprintf("property-slow-%d-%d", propertyID, taskID), nil
						default:
							return fmt.Sprintf("property-default-%d-%d", propertyID, taskID), nil
						}
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("property-task-%d-%d", propertyID, taskID))
					workflow := Setup(taskInstance)

					result, err := workflow.Await()

					// Property verification: Result should be consistent with error state
					if err != nil && result != nil && !result.HasErrors() {
						atomic.AddInt64(&propertyFailures, 1)
						t.Errorf("Property violation: Error returned but result has no errors")
					}

					if err == nil && result != nil && result.HasErrors() {
						atomic.AddInt64(&propertyFailures, 1)
						t.Errorf("Property violation: No error returned but result has errors")
					}
				}(j)
			}

			taskWg.Wait()
			atomic.AddInt64(&propertyTests, 1)
		}(i)
	}

	wg.Wait()

	if propertyTests != numProperties {
		t.Errorf("Expected %d property tests, completed %d", numProperties, propertyTests)
	}

	if propertyFailures > 0 {
		t.Errorf("Property-based testing found %d violations", propertyFailures)
	}

	t.Logf("Property-based testing completed: %d tests, %d failures", propertyTests, propertyFailures)
}
