package orchestrator

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/maniartech/orchestrator"
	"github.com/maniartech/orchestrator/internal/task"
)

// GoroutineTracker helps track goroutine leaks
type GoroutineTracker struct {
	initial int
	peak    int
	current int
	samples []int
	mu      sync.RWMutex
}

// NewGoroutineTracker creates a new goroutine tracker
func NewGoroutineTracker() *GoroutineTracker {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	initial := runtime.NumGoroutine()

	return &GoroutineTracker{
		initial: initial,
		peak:    initial,
		current: initial,
		samples: make([]int, 0, 100),
	}
}

// Sample records the current goroutine count
func (gt *GoroutineTracker) Sample() {
	current := runtime.NumGoroutine()

	gt.mu.Lock()
	defer gt.mu.Unlock()

	gt.current = current
	gt.samples = append(gt.samples, current)

	if current > gt.peak {
		gt.peak = current
	}
}

// Report returns a summary of goroutine usage
func (gt *GoroutineTracker) Report() (initial, peak, final, growth int, samples []int) {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	return gt.initial, gt.peak, gt.current, gt.current - gt.initial, append([]int(nil), gt.samples...)
}

// TestGoroutineLeakDetection_BasicTasks tests basic task execution for leaks
func TestGoroutineLeakDetection_BasicTasks(t *testing.T) {
	tracker := NewGoroutineTracker()

	const numTasks = 100
	var wg sync.WaitGroup
	var completedTasks int64

	wg.Add(numTasks)
	for i := 0; i < numTasks; i++ {
		go func(taskID int) {
			defer wg.Done()
			defer func() {
				atomic.AddInt64(&completedTasks, 1)
			}()

			taskFn := func() (string, error) {
				time.Sleep(time.Microsecond)
				return fmt.Sprintf("basic-task-%d", taskID), nil
			}

			taskInstance := task.Task(taskFn).Named(fmt.Sprintf("basic-task-%d", taskID))
			workflow := Setup(taskInstance)

			_, err := workflow.Await()
			if err != nil {
				t.Errorf("Basic task %d failed: %v", taskID, err)
			}
		}(i)
	}

	// Sample during execution
	go func() {
		for atomic.LoadInt64(&completedTasks) < numTasks {
			tracker.Sample()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Final cleanup and measurement
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	tracker.Sample()

	initial, peak, final, growth, samples := tracker.Report()

	t.Logf("Basic tasks goroutine leak test:")
	t.Logf("  Tasks completed: %d/%d", atomic.LoadInt64(&completedTasks), numTasks)
	t.Logf("  Initial goroutines: %d", initial)
	t.Logf("  Peak goroutines: %d", peak)
	t.Logf("  Final goroutines: %d", final)
	t.Logf("  Growth: %d", growth)
	// Show last few samples if available
	if len(samples) > 0 {
		start := len(samples) - 5
		if start < 0 {
			start = 0
		}
		t.Logf("  Samples: %v", samples[start:])
	}

	if atomic.LoadInt64(&completedTasks) != numTasks {
		t.Errorf("Not all tasks completed: %d/%d", atomic.LoadInt64(&completedTasks), numTasks)
	}

	maxAcceptableGrowth := 10
	if growth > maxAcceptableGrowth {
		t.Errorf("Goroutine leak detected: %d growth (max acceptable: %d)", growth, maxAcceptableGrowth)
	}
}

// TestGoroutineLeakDetection_NestedGoroutines tests nested goroutine scenarios
func TestGoroutineLeakDetection_NestedGoroutines(t *testing.T) {
	tracker := NewGoroutineTracker()

	const numTasks = 50
	const goroutinesPerTask = 5

	var wg sync.WaitGroup
	var completedTasks int64
	var totalNestedGoroutines int64

	wg.Add(numTasks)
	for i := 0; i < numTasks; i++ {
		go func(taskID int) {
			defer wg.Done()
			defer func() {
				atomic.AddInt64(&completedTasks, 1)
			}()

			taskFn := func() ([]string, error) {
				var nestedWg sync.WaitGroup
				results := make([]string, goroutinesPerTask)

				nestedWg.Add(goroutinesPerTask)
				for j := 0; j < goroutinesPerTask; j++ {
					go func(idx int) {
						defer nestedWg.Done()
						atomic.AddInt64(&totalNestedGoroutines, 1)

						// Simulate work
						time.Sleep(time.Duration(idx+1) * time.Microsecond)
						results[idx] = fmt.Sprintf("nested-%d-%d", taskID, idx)
					}(j)
				}

				nestedWg.Wait()
				return results, nil
			}

			taskInstance := task.Task(taskFn).Named(fmt.Sprintf("nested-task-%d", taskID))
			workflow := Setup(taskInstance)

			result, err := workflow.Await()
			if err != nil {
				t.Errorf("Nested task %d failed: %v", taskID, err)
				return
			}

			// Verify results
			if result != nil {
				value := result.Get(fmt.Sprintf("nested-task-%d", taskID))
				if results, ok := value.([]string); ok {
					if len(results) != goroutinesPerTask {
						t.Errorf("Expected %d results, got %d", goroutinesPerTask, len(results))
					}
				}
			}
		}(i)
	}

	// Sample during execution
	go func() {
		for atomic.LoadInt64(&completedTasks) < numTasks {
			tracker.Sample()
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Final cleanup and measurement
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	tracker.Sample()

	initial, peak, final, growth, _ := tracker.Report()
	expectedNestedGoroutines := int64(numTasks * goroutinesPerTask)

	t.Logf("Nested goroutines leak test:")
	t.Logf("  Tasks completed: %d/%d", atomic.LoadInt64(&completedTasks), numTasks)
	t.Logf("  Nested goroutines created: %d/%d", atomic.LoadInt64(&totalNestedGoroutines), expectedNestedGoroutines)
	t.Logf("  Initial goroutines: %d", initial)
	t.Logf("  Peak goroutines: %d", peak)
	t.Logf("  Final goroutines: %d", final)
	t.Logf("  Growth: %d", growth)

	if atomic.LoadInt64(&completedTasks) != numTasks {
		t.Errorf("Not all tasks completed: %d/%d", atomic.LoadInt64(&completedTasks), numTasks)
	}

	if atomic.LoadInt64(&totalNestedGoroutines) != expectedNestedGoroutines {
		t.Errorf("Not all nested goroutines created: %d/%d", atomic.LoadInt64(&totalNestedGoroutines), expectedNestedGoroutines)
	}

	maxAcceptableGrowth := 15
	if growth > maxAcceptableGrowth {
		t.Errorf("Goroutine leak detected: %d growth (max acceptable: %d)", growth, maxAcceptableGrowth)
	}
}

// TestGoroutineLeakDetection_LongRunningTasks tests long-running task scenarios
func TestGoroutineLeakDetection_LongRunningTasks(t *testing.T) {
	tracker := NewGoroutineTracker()

	const (
		testDuration = 3 * time.Second
		numWorkers   = 10
		taskDuration = 100 * time.Millisecond
	)

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	var wg sync.WaitGroup
	var completedTasks int64
	var cancelledTasks int64

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			taskID := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					taskFn := func() (string, error) {
						// Long-running task that can be cancelled
						select {
						case <-ctx.Done():
							return "", ctx.Err()
						case <-time.After(taskDuration):
							return fmt.Sprintf("long-task-%d-%d", workerID, taskID), nil
						}
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("long-task-%d-%d", workerID, taskID))
					workflow := Setup(taskInstance)

					result, err := workflow.AwaitWithContext(ctx)

					if err != nil {
						if err == context.DeadlineExceeded || err == context.Canceled {
							atomic.AddInt64(&cancelledTasks, 1)
						} else {
							t.Errorf("Long task failed with unexpected error: %v", err)
						}
					} else if result != nil {
						atomic.AddInt64(&completedTasks, 1)
					}

					taskID++
				}
			}
		}(i)
	}

	// Sample during execution
	sampleTicker := time.NewTicker(100 * time.Millisecond)
	defer sampleTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sampleTicker.C:
				tracker.Sample()
			}
		}
	}()

	wg.Wait()

	// Final cleanup and measurement
	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	tracker.Sample()

	initial, peak, final, growth, samples := tracker.Report()

	t.Logf("Long-running tasks leak test:")
	t.Logf("  Duration: %v", testDuration)
	t.Logf("  Completed tasks: %d", atomic.LoadInt64(&completedTasks))
	t.Logf("  Cancelled tasks: %d", atomic.LoadInt64(&cancelledTasks))
	t.Logf("  Initial goroutines: %d", initial)
	t.Logf("  Peak goroutines: %d", peak)
	t.Logf("  Final goroutines: %d", final)
	t.Logf("  Growth: %d", growth)
	t.Logf("  Sample count: %d", len(samples))

	totalTasks := atomic.LoadInt64(&completedTasks) + atomic.LoadInt64(&cancelledTasks)
	if totalTasks == 0 {
		t.Error("No tasks completed or cancelled")
	}

	maxAcceptableGrowth := 20
	if growth > maxAcceptableGrowth {
		t.Errorf("Goroutine leak detected: %d growth (max acceptable: %d)", growth, maxAcceptableGrowth)
	}
}

// TestGoroutineLeakDetection_PanicRecovery tests panic scenarios for leaks
func TestGoroutineLeakDetection_PanicRecovery(t *testing.T) {
	tracker := NewGoroutineTracker()

	const numTasks = 50
	const panicRate = 0.3 // 30% of tasks will panic

	var wg sync.WaitGroup
	var completedTasks int64
	var panicTasks int64

	wg.Add(numTasks)
	for i := 0; i < numTasks; i++ {
		go func(taskID int) {
			defer wg.Done()

			taskFn := func() (string, error) {
				// Simulate panic in some tasks
				if float64(taskID%10)/10.0 < panicRate {
					panic(fmt.Sprintf("simulated panic in task %d", taskID))
				}

				time.Sleep(time.Microsecond)
				return fmt.Sprintf("panic-test-task-%d", taskID), nil
			}

			taskInstance := task.Task(taskFn).Named(fmt.Sprintf("panic-test-task-%d", taskID))
			workflow := Setup(taskInstance)

			result, err := workflow.Await()

			if err != nil || (result != nil && result.HasErrors()) {
				// Check if it was a panic
				if result != nil && result.HasErrors() {
					errors := result.Errors()
					for _, opErr := range errors {
						if opErr.Stack != nil {
							atomic.AddInt64(&panicTasks, 1)
							return
						}
					}
				}
				t.Errorf("Task %d failed unexpectedly: %v", taskID, err)
			} else {
				atomic.AddInt64(&completedTasks, 1)
			}
		}(i)
	}

	// Sample during execution
	go func() {
		for atomic.LoadInt64(&completedTasks)+atomic.LoadInt64(&panicTasks) < numTasks {
			tracker.Sample()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Final cleanup and measurement
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	tracker.Sample()

	initial, peak, final, growth, _ := tracker.Report()

	t.Logf("Panic recovery leak test:")
	t.Logf("  Tasks completed: %d", atomic.LoadInt64(&completedTasks))
	t.Logf("  Tasks panicked: %d", atomic.LoadInt64(&panicTasks))
	t.Logf("  Total processed: %d/%d", atomic.LoadInt64(&completedTasks)+atomic.LoadInt64(&panicTasks), numTasks)
	t.Logf("  Initial goroutines: %d", initial)
	t.Logf("  Peak goroutines: %d", peak)
	t.Logf("  Final goroutines: %d", final)
	t.Logf("  Growth: %d", growth)

	totalProcessed := atomic.LoadInt64(&completedTasks) + atomic.LoadInt64(&panicTasks)
	if totalProcessed != numTasks {
		t.Errorf("Not all tasks processed: %d/%d", totalProcessed, numTasks)
	}

	// Verify some panics were recovered
	if atomic.LoadInt64(&panicTasks) == 0 {
		t.Error("Expected some panics to be recovered, got none")
	}

	maxAcceptableGrowth := 15
	if growth > maxAcceptableGrowth {
		t.Errorf("Goroutine leak detected: %d growth (max acceptable: %d)", growth, maxAcceptableGrowth)
	}
}

// TestGoroutineLeakDetection_ChannelOperations tests channel-based operations
func TestGoroutineLeakDetection_ChannelOperations(t *testing.T) {
	tracker := NewGoroutineTracker()

	const (
		numProducers        = 10
		numConsumers        = 5
		messagesPerProducer = 20
		channelBuffer       = 100
	)

	ch := make(chan string, channelBuffer)
	var wg sync.WaitGroup
	var producedMessages int64
	var consumedMessages int64

	// Start consumers
	wg.Add(numConsumers)
	for i := 0; i < numConsumers; i++ {
		go func(consumerID int) {
			defer wg.Done()

			for msg := range ch {
				atomic.AddInt64(&consumedMessages, 1)

				// Process message using orchestrator
				taskFn := func() (string, error) {
					return fmt.Sprintf("processed-%s-by-consumer-%d", msg, consumerID), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("consumer-task-%d", consumerID))
				workflow := Setup(taskInstance)

				_, err := workflow.Await()
				if err != nil {
					t.Errorf("Consumer task failed: %v", err)
				}
			}
		}(i)
	}

	// Start producers
	var producerWg sync.WaitGroup
	producerWg.Add(numProducers)
	for i := 0; i < numProducers; i++ {
		go func(producerID int) {
			defer producerWg.Done()

			for j := 0; j < messagesPerProducer; j++ {
				msg := fmt.Sprintf("msg-%d-%d", producerID, j)

				// Produce message using orchestrator
				taskFn := func() (string, error) {
					ch <- msg
					atomic.AddInt64(&producedMessages, 1)
					return msg, nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("producer-task-%d-%d", producerID, j))
				workflow := Setup(taskInstance)

				_, err := workflow.Await()
				if err != nil {
					t.Errorf("Producer task failed: %v", err)
				}
			}
		}(i)
	}

	// Sample during execution
	go func() {
		for atomic.LoadInt64(&producedMessages) < int64(numProducers*messagesPerProducer) ||
			atomic.LoadInt64(&consumedMessages) < atomic.LoadInt64(&producedMessages) {
			tracker.Sample()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Wait for producers to finish
	producerWg.Wait()
	close(ch)

	// Wait for consumers to finish
	wg.Wait()

	// Final cleanup and measurement
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	tracker.Sample()

	initial, peak, final, growth, _ := tracker.Report()
	expectedMessages := int64(numProducers * messagesPerProducer)

	t.Logf("Channel operations leak test:")
	t.Logf("  Produced messages: %d/%d", atomic.LoadInt64(&producedMessages), expectedMessages)
	t.Logf("  Consumed messages: %d/%d", atomic.LoadInt64(&consumedMessages), expectedMessages)
	t.Logf("  Initial goroutines: %d", initial)
	t.Logf("  Peak goroutines: %d", peak)
	t.Logf("  Final goroutines: %d", final)
	t.Logf("  Growth: %d", growth)

	if atomic.LoadInt64(&producedMessages) != expectedMessages {
		t.Errorf("Not all messages produced: %d/%d", atomic.LoadInt64(&producedMessages), expectedMessages)
	}

	if atomic.LoadInt64(&consumedMessages) != expectedMessages {
		t.Errorf("Not all messages consumed: %d/%d", atomic.LoadInt64(&consumedMessages), expectedMessages)
	}

	maxAcceptableGrowth := 20
	if growth > maxAcceptableGrowth {
		t.Errorf("Goroutine leak detected: %d growth (max acceptable: %d)", growth, maxAcceptableGrowth)
	}
}

// TestGoroutineLeakDetection_ContextCancellation tests context cancellation scenarios
func TestGoroutineLeakDetection_ContextCancellation(t *testing.T) {
	tracker := NewGoroutineTracker()

	const (
		numIterations     = 20
		tasksPerIteration = 10
		cancellationDelay = 50 * time.Millisecond
	)

	var totalTasks int64
	var cancelledTasks int64
	var completedTasks int64

	for i := 0; i < numIterations; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		var wg sync.WaitGroup
		wg.Add(tasksPerIteration)

		for j := 0; j < tasksPerIteration; j++ {
			atomic.AddInt64(&totalTasks, 1)

			go func(iteration, taskID int) {
				defer wg.Done()

				taskFn := func() (string, error) {
					// Long-running task that can be cancelled
					for k := 0; k < 100; k++ {
						select {
						case <-ctx.Done():
							return "", ctx.Err()
						default:
							time.Sleep(time.Millisecond)
						}
					}
					return fmt.Sprintf("context-task-%d-%d", iteration, taskID), nil
				}

				taskInstance := task.Task(taskFn).Named(fmt.Sprintf("context-task-%d-%d", iteration, taskID))
				workflow := Setup(taskInstance)

				result, err := workflow.AwaitWithContext(ctx)

				if err != nil {
					if err == context.Canceled {
						atomic.AddInt64(&cancelledTasks, 1)
					} else {
						t.Errorf("Task failed with unexpected error: %v", err)
					}
				} else if result != nil {
					atomic.AddInt64(&completedTasks, 1)
				}
			}(i, j)
		}

		// Cancel after delay
		go func() {
			time.Sleep(cancellationDelay)
			cancel()
		}()

		wg.Wait()

		// Sample after each iteration
		if i%5 == 0 {
			tracker.Sample()
		}
	}

	// Final cleanup and measurement
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	tracker.Sample()

	initial, peak, final, growth, _ := tracker.Report()

	t.Logf("Context cancellation leak test:")
	t.Logf("  Total tasks: %d", atomic.LoadInt64(&totalTasks))
	t.Logf("  Cancelled tasks: %d", atomic.LoadInt64(&cancelledTasks))
	t.Logf("  Completed tasks: %d", atomic.LoadInt64(&completedTasks))
	t.Logf("  Initial goroutines: %d", initial)
	t.Logf("  Peak goroutines: %d", peak)
	t.Logf("  Final goroutines: %d", final)
	t.Logf("  Growth: %d", growth)

	totalProcessed := atomic.LoadInt64(&cancelledTasks) + atomic.LoadInt64(&completedTasks)
	if totalProcessed != atomic.LoadInt64(&totalTasks) {
		t.Errorf("Not all tasks processed: %d/%d", totalProcessed, atomic.LoadInt64(&totalTasks))
	}

	// Should have some cancellations
	if atomic.LoadInt64(&cancelledTasks) == 0 {
		t.Error("Expected some cancellations, got none")
	}

	maxAcceptableGrowth := 25
	if growth > maxAcceptableGrowth {
		t.Errorf("Goroutine leak detected: %d growth (max acceptable: %d)", growth, maxAcceptableGrowth)
	}
}
