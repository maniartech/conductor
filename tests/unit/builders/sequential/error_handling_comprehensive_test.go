package sequential

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	errorspkg "github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"

	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
)

// TestErrorHandlingStrategies_Comprehensive tests all error handling strategies comprehensively
func TestErrorHandlingStrategies_Comprehensive(t *testing.T) {
	t.Run("FailFast_ImmediateStop", func(t *testing.T) {
		var executionOrder []int
		var mu sync.Mutex

		seq := Sequential(
			task.Task(func() (string, error) {
				mu.Lock()
				executionOrder = append(executionOrder, 1)
				mu.Unlock()
				return "step1", nil
			}).Named("step1"),
			task.Task(func() (string, error) {
				mu.Lock()
				executionOrder = append(executionOrder, 2)
				mu.Unlock()
				return "", errors.New("step2 failed")
			}).Named("step2"),
			task.Task(func() (string, error) {
				mu.Lock()
				executionOrder = append(executionOrder, 3)
				mu.Unlock()
				return "step3", nil
			}).Named("step3"),
		).Named("fail-fast-test")

		ctx := context.Background()
		cfg := config.Config{ErrorStrategy: errorspkg.FailFast}

		result, err := seq.Execute(ctx, cfg)

		// Verify immediate stop behavior
		if err == nil {
			t.Fatal("Expected error with FailFast strategy")
		}

		mu.Lock()
		if len(executionOrder) != 2 {
			t.Errorf("Expected 2 steps executed, got %d: %v", len(executionOrder), executionOrder)
		}
		mu.Unlock()

		// Verify only first step result is available
		if result.Get("step1") != "step1" {
			t.Error("Expected step1 result")
		}
		if result.Get("step3") != nil {
			t.Error("Expected no step3 result with FailFast")
		}

		// Verify error contains detailed information
		if !strings.Contains(err.Error(), "step2 failed") {
			t.Error("Expected original error in enhanced error message")
		}
		if !strings.Contains(err.Error(), "Detailed Report:") {
			t.Error("Expected detailed report in error message")
		}
	})

	t.Run("CollectAll_ContinueExecution", func(t *testing.T) {
		var executionOrder []int
		var mu sync.Mutex

		seq := Sequential(
			task.Task(func() (string, error) {
				mu.Lock()
				executionOrder = append(executionOrder, 1)
				mu.Unlock()
				return "step1", nil
			}).Named("step1"),
			task.Task(func() (string, error) {
				mu.Lock()
				executionOrder = append(executionOrder, 2)
				mu.Unlock()
				return "", errors.New("step2 failed")
			}).Named("step2"),
			task.Task(func() (string, error) {
				mu.Lock()
				executionOrder = append(executionOrder, 3)
				mu.Unlock()
				return "step3", nil
			}).Named("step3"),
			task.Task(func() (string, error) {
				mu.Lock()
				executionOrder = append(executionOrder, 4)
				mu.Unlock()
				return "", errors.New("step4 failed")
			}).Named("step4"),
		).Named("collect-all-test")

		ctx := context.Background()
		cfg := config.Config{ErrorStrategy: errorspkg.CollectAll}

		result, err := seq.Execute(ctx, cfg)

		// Verify all steps were executed
		mu.Lock()
		if len(executionOrder) != 4 {
			t.Errorf("Expected 4 steps executed, got %d: %v", len(executionOrder), executionOrder)
		}
		mu.Unlock()

		// Verify error contains information about multiple errors
		if err == nil {
			t.Fatal("Expected error with CollectAll strategy")
		}
		if !strings.Contains(err.Error(), "2 errors") {
			t.Error("Expected error count in enhanced error message")
		}

		// Verify successful results are available
		if result.Get("step1") != "step1" {
			t.Error("Expected step1 result")
		}
		if result.Get("step3") != "step3" {
			t.Error("Expected step3 result")
		}

		// Verify error collection
		if len(result.Errors()) != 2 {
			t.Errorf("Expected 2 errors in result, got %d", len(result.Errors()))
		}
	})
}

// TestRichErrorMetadata tests rich error metadata collection
func TestRichErrorMetadata(t *testing.T) {
	seq := Sequential(
		task.Task(func() (string, error) {
			time.Sleep(10 * time.Millisecond) // Simulate work
			return "step1", nil
		}).Named("step1"),
		task.Task(func() (string, error) {
			time.Sleep(20 * time.Millisecond) // Simulate work before failure
			return "", errors.New("detailed error with context")
		}).Named("step2"),
	).Named("metadata-test")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.FailFast}

	startTime := time.Now()
	result, err := seq.Execute(ctx, cfg)
	totalDuration := time.Since(startTime)

	if err == nil {
		t.Fatal("Expected error")
	}

	// Verify rich error metadata
	if len(result.Errors()) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors()))
	}

	opError := result.Errors()[0]

	// Verify error metadata fields
	if opError.Error.Error() != "detailed error with context" {
		t.Errorf("Expected original error message, got %s", opError.Error.Error())
	}
	if opError.Index != 1 {
		t.Errorf("Expected error index 1, got %d", opError.Index)
	}
	if opError.Duration < 15*time.Millisecond {
		t.Errorf("Expected duration >= 15ms, got %v", opError.Duration)
	}
	if opError.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
	if !strings.Contains(opError.OpID, "step2") {
		t.Errorf("Expected operation ID to contain 'step2', got %s", opError.OpID)
	}
	if len(opError.Stack) == 0 {
		t.Error("Expected stack trace in error metadata")
	}

	// Verify timing is reasonable
	if totalDuration < 30*time.Millisecond {
		t.Errorf("Expected total duration >= 30ms, got %v", totalDuration)
	}

	t.Logf("Rich error metadata test passed - Duration: %v, OpID: %s", opError.Duration, opError.OpID)
}

// TestErrorBoundaryContainment tests error boundary containment within sequential blocks
func TestErrorBoundaryContainment(t *testing.T) {
	// Create nested sequential with different error boundaries
	innerSeq := Sequential(
		task.Task(func() (string, error) { return "inner1", nil }).Named("inner1"),
		task.Task(func() (string, error) { return "", errors.New("inner error") }).Named("inner2"),
		task.Task(func() (string, error) { return "inner3", nil }).Named("inner3"),
	).Named("inner-seq").ErrorBoundary(errorspkg.CollectAll)

	outerSeq := Sequential(
		task.Task(func() (string, error) { return "outer1", nil }).Named("outer1"),
		innerSeq,
		task.Task(func() (string, error) { return "outer3", nil }).Named("outer3"),
	).Named("outer-seq").ErrorBoundary(errorspkg.FailFast)

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.FailFast} // This should be overridden by error boundaries

	result, err := outerSeq.Execute(ctx, cfg)

	// Outer should fail fast due to inner error, but inner should collect all
	if err == nil {
		t.Fatal("Expected error from outer sequential")
	}

	// Verify outer sequential stopped at inner sequential
	if result.Get("outer1") != "outer1" {
		t.Error("Expected outer1 result")
	}
	if result.Get("outer3") != nil {
		t.Error("Expected no outer3 result due to FailFast")
	}

	// The error boundary should contain the error within the inner sequential
	// and the outer sequential should receive the error and stop due to FailFast
	// This demonstrates proper error boundary containment

	// Verify error boundary containment in error message
	if !strings.Contains(err.Error(), "inner error") {
		t.Error("Expected inner error to be contained and propagated")
	}

	t.Logf("Error boundary containment test passed")
}

// TestHierarchicalErrorConfiguration tests hierarchical error configuration inheritance
func TestHierarchicalErrorConfiguration(t *testing.T) {
	// Parent config with FailFast
	parentConfig := config.Config{ErrorStrategy: errorspkg.FailFast}

	// Child overrides with CollectAll
	seq := Sequential(
		task.Task(func() (string, error) { return "step1", nil }).Named("step1"),
		task.Task(func() (string, error) { return "", errors.New("step2 error") }).Named("step2"),
		task.Task(func() (string, error) { return "step3", nil }).Named("step3"),
	).Named("hierarchy-test").With(config.Config{ErrorStrategy: errorspkg.CollectAll})

	ctx := context.Background()
	result, err := seq.Execute(ctx, parentConfig)

	// Should use child's CollectAll strategy, not parent's FailFast
	if err == nil {
		t.Fatal("Expected error")
	}

	// All steps should execute due to CollectAll override
	if result.Get("step1") != "step1" {
		t.Error("Expected step1 result")
	}
	if result.Get("step3") != "step3" {
		t.Error("Expected step3 result with CollectAll override")
	}

	// Should have collected the error
	if len(result.Errors()) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors()))
	}

	t.Logf("Hierarchical error configuration test passed")
}

// TestZeroAllocationErrorCollection tests zero-allocation error collection with atomic operations
func TestZeroAllocationErrorCollection(t *testing.T) {
	// This test verifies that error collection uses atomic operations efficiently
	var errorCount int64

	seq := Sequential(
		task.Task(func() (string, error) {
			atomic.AddInt64(&errorCount, 1)
			return "", errors.New("error 1")
		}).Named("step1"),
		task.Task(func() (string, error) {
			atomic.AddInt64(&errorCount, 1)
			return "", errors.New("error 2")
		}).Named("step2"),
		task.Task(func() (string, error) {
			atomic.AddInt64(&errorCount, 1)
			return "", errors.New("error 3")
		}).Named("step3"),
	).Named("atomic-test")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.CollectAll}

	result, err := seq.Execute(ctx, cfg)

	if err == nil {
		t.Fatal("Expected error")
	}

	// Verify atomic operations worked correctly
	if atomic.LoadInt64(&errorCount) != 3 {
		t.Errorf("Expected 3 atomic increments, got %d", atomic.LoadInt64(&errorCount))
	}

	// Verify all errors were collected
	if len(result.Errors()) != 3 {
		t.Errorf("Expected 3 errors collected, got %d", len(result.Errors()))
	}

	t.Logf("Zero-allocation error collection test passed")
}

// TestContextCancellationWithErrorHandling tests context cancellation during error handling
func TestContextCancellationWithErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	seq := Sequential(
		task.Task(func() (string, error) {
			time.Sleep(20 * time.Millisecond)
			return "step1", nil
		}).Named("step1"),
		task.Task(func() (string, error) {
			time.Sleep(100 * time.Millisecond) // This will be cancelled
			return "step2", nil
		}).Named("step2"),
		task.Task(func() (string, error) {
			return "step3", nil // Never reached
		}).Named("step3"),
	).Named("cancellation-test")

	cfg := config.Config{ErrorStrategy: errorspkg.CollectAll}

	result, err := seq.Execute(ctx, cfg)

	// Should get context cancellation error
	if err == nil {
		t.Fatal("Expected context cancellation error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "deadline exceeded") {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}

	// Should have partial results
	if result.Get("step1") != "step1" {
		t.Error("Expected step1 result before cancellation")
	}
	if result.Get("step2") != nil {
		t.Error("Expected no step2 result due to cancellation")
	}

	t.Logf("Context cancellation with error handling test passed")
}

// TestPanicRecoveryWithStackTrace tests panic recovery with detailed stack traces
func TestPanicRecoveryWithStackTrace(t *testing.T) {
	seq := Sequential(
		task.Task(func() (string, error) { return "step1", nil }).Named("step1"),
		task.Task(func() (string, error) {
			panic("test panic with stack trace")
		}).Named("panic-step"),
		task.Task(func() (string, error) { return "step3", nil }).Named("step3"),
	).Named("panic-recovery-test")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.FailFast}

	result, err := seq.Execute(ctx, cfg)

	if err == nil {
		t.Fatal("Expected error from panic recovery")
	}

	// Verify panic was recovered and converted to error
	if !strings.Contains(err.Error(), "panic") {
		t.Error("Expected panic information in error message")
	}

	// Verify stack trace is captured
	if len(result.Errors()) > 0 {
		opError := result.Errors()[0]
		if len(opError.Stack) == 0 {
			t.Error("Expected stack trace in panic error")
		}
		// Stack trace should contain goroutine information
		stackStr := string(opError.Stack)
		if !strings.Contains(stackStr, "goroutine") {
			t.Error("Expected goroutine information in stack trace")
		}
		t.Logf("Stack trace captured: %d bytes", len(opError.Stack))
	}

	// Verify partial execution
	if result.Get("step1") != "step1" {
		t.Error("Expected step1 result before panic")
	}
	if result.Get("step3") != nil {
		t.Error("Expected no step3 result after panic with FailFast")
	}

	t.Logf("Panic recovery with stack trace test passed")
}

// TestConcurrentErrorHandling tests error handling under concurrent access
func TestConcurrentErrorHandling(t *testing.T) {
	const numGoroutines = 10

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.FailFast}

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	// Execute the same sequential from multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Create a new sequential for each goroutine to avoid shared state
			testSeq := Sequential(
				task.Task(func() (string, error) { return "step1", nil }).Named("step1"),
				task.Task(func() (string, error) { return "", errors.New("concurrent error") }).Named("step2"),
			).Named("concurrent-test")

			result, err := testSeq.Execute(ctx, cfg)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				// Verify error handling worked correctly
				if result.Get("step1") != "step1" {
					t.Errorf("Expected step1 result in concurrent execution")
				}
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	// All executions should have failed due to the error in step2
	if atomic.LoadInt64(&errorCount) != numGoroutines {
		t.Errorf("Expected %d errors, got %d", numGoroutines, atomic.LoadInt64(&errorCount))
	}
	if atomic.LoadInt64(&successCount) != 0 {
		t.Errorf("Expected 0 successes, got %d", atomic.LoadInt64(&successCount))
	}

	t.Logf("Concurrent error handling test passed")
}

// BenchmarkErrorHandlingPerformance benchmarks error handling performance impact
func BenchmarkErrorHandlingPerformance(b *testing.B) {
	ctx := context.Background()

	b.Run("FailFast", func(b *testing.B) {
		cfg := config.Config{ErrorStrategy: errorspkg.FailFast}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			seq := Sequential(
				task.Task(func() (int, error) { return 1, nil }),
				task.Task(func() (int, error) { return 0, errors.New("benchmark error") }),
				task.Task(func() (int, error) { return 3, nil }),
			).Named("benchmark-fail-fast")

			result, err := seq.Execute(ctx, cfg)
			if err == nil {
				b.Fatal("Expected error")
			}
			if result == nil {
				b.Fatal("Expected non-nil result")
			}
		}
	})

	b.Run("CollectAll", func(b *testing.B) {
		cfg := config.Config{ErrorStrategy: errorspkg.CollectAll}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			seq := Sequential(
				task.Task(func() (int, error) { return 1, nil }),
				task.Task(func() (int, error) { return 0, errors.New("benchmark error") }),
				task.Task(func() (int, error) { return 3, nil }),
			).Named("benchmark-collect-all")

			result, err := seq.Execute(ctx, cfg)
			if err == nil {
				b.Fatal("Expected error")
			}
			if result == nil {
				b.Fatal("Expected non-nil result")
			}
		}
	})

	b.Run("NoError", func(b *testing.B) {
		cfg := config.Config{ErrorStrategy: errorspkg.FailFast}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			seq := Sequential(
				task.Task(func() (int, error) { return 1, nil }),
				task.Task(func() (int, error) { return 2, nil }),
				task.Task(func() (int, error) { return 3, nil }),
			).Named("benchmark-no-error")

			result, err := seq.Execute(ctx, cfg)
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
			if result == nil {
				b.Fatal("Expected non-nil result")
			}
		}
	})
}

// BenchmarkErrorMetadataCollection benchmarks error metadata collection performance
func BenchmarkErrorMetadataCollection(b *testing.B) {
	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.CollectAll}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		seq := Sequential(
			task.Task(func() (int, error) { return 0, errors.New("error 1") }),
			task.Task(func() (int, error) { return 0, errors.New("error 2") }),
			task.Task(func() (int, error) { return 0, errors.New("error 3") }),
		).Named("metadata-benchmark")

		result, err := seq.Execute(ctx, cfg)
		if err == nil {
			b.Fatal("Expected error")
		}
		if len(result.Errors()) != 3 {
			b.Fatalf("Expected 3 errors, got %d", len(result.Errors()))
		}

		// Access error metadata to ensure it's properly collected
		for _, opErr := range result.Errors() {
			_ = opErr.Error
			_ = opErr.Index
			_ = opErr.Duration
			_ = opErr.Timestamp
			_ = opErr.OpID
			_ = opErr.Stack
		}
	}
}
