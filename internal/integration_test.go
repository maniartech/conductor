// Package internal provides integration tests for all infrastructure components
package internal

import (
	systemContext "context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/pool"
	"github.com/maniartech/orchestrator/internal/status"
	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/context"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/result"
)

// TestInfrastructureIntegration tests all infrastructure components working together
func TestInfrastructureIntegration(t *testing.T) {
	// Create pool manager
	poolManager := pool.NewManager()

	// Create status manager
	statusManager := status.NewManager()

	// Create configuration
	config := config.DefaultConfig()
	config.MaxConcurrency = 10
	config.Timeout = 5 * time.Second

	// Create context
	ctx := context.NewContext(config)

	// Create result container
	result := result.NewResult()

	// Test integration scenario
	const numOperations = 100
	var wg sync.WaitGroup
	wg.Add(numOperations)

	// Simulate concurrent orchestration operations
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()

			// Get orchestrator item from pool
			orchItem := poolManager.GetOrchestrator()
			defer poolManager.PutOrchestrator(orchItem)

			// Transition status
			if !statusManager.TransitionToRunning() {
				// If already running, that's fine for this test
			}

			// Set context value
			ctx.Set(fmt.Sprintf("operation-%d", id), id)

			// Get slice from pool
			slice := poolManager.GetSlice(10)
			defer poolManager.PutSlice(slice)

			// Simulate work
			slice.Data = append(slice.Data, id)

			// Store result
			result.Set(fmt.Sprintf("result-%d", id), id*2)

			// Get context item from pool
			ctxItem := poolManager.GetContext()
			defer poolManager.PutContext(ctxItem)

			ctxItem.Values["test"] = id

			// Get result item from pool
			resultItem := poolManager.GetResult()
			defer poolManager.PutResult(resultItem)

			resultItem.Entries["test"] = id
		}(i)
	}

	wg.Wait()

	// Verify final state
	if !statusManager.IsRunning() && !statusManager.IsCompleted() {
		t.Error("Expected status to be Running or Completed")
	}

	// Check that results were stored
	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("result-%d", i)
		if value := result.Get(key); value != i*2 {
			t.Errorf("Expected result %d to be %d, got %v", i, i*2, value)
		}
	}

	// Check pool statistics
	stats := poolManager.Stats()
	if stats.OrchestratorGets != int64(numOperations) {
		t.Errorf("Expected %d orchestrator gets, got %d", numOperations, stats.OrchestratorGets)
	}

	if stats.OrchestratorPuts != int64(numOperations) {
		t.Errorf("Expected %d orchestrator puts, got %d", numOperations, stats.OrchestratorPuts)
	}
}

// TestConfigurationInheritance tests hierarchical configuration
func TestConfigurationInheritance(t *testing.T) {
	parent := config.Config{
		ErrorStrategy:  errors.FailFast,
		Timeout:        10 * time.Second,
		Retries:        3,
		MaxConcurrency: 50,
		Context:        systemContext.Background(),
	}

	child := config.Config{
		ErrorStrategy: errors.CollectAll,
		Timeout:       5 * time.Second,
		// Retries and MaxConcurrency should inherit from parent
	}

	result := child.Inherit(parent)

	// Verify inheritance
	if result.ErrorStrategy != errors.CollectAll {
		t.Error("Child should override ErrorStrategy")
	}

	if result.Timeout != 5*time.Second {
		t.Error("Child should override Timeout")
	}

	if result.Retries != 3 {
		t.Error("Child should inherit Retries from parent")
	}

	if result.MaxConcurrency != 50 {
		t.Error("Child should inherit MaxConcurrency from parent")
	}
}

// TestErrorHandlingIntegration tests error handling across components
func TestErrorHandlingIntegration(t *testing.T) {
	result := result.NewResult()

	// Add multiple errors
	for i := 0; i < 5; i++ {
		opErr := errors.OperationError{
			Error:     fmt.Errorf("error %d", i),
			Index:     i,
			Duration:  time.Duration(i) * time.Millisecond,
			Timestamp: time.Now(),
			OpID:      fmt.Sprintf("op-%d", i),
		}
		result.AddError(opErr)
	}

	// Verify error collection
	if !result.HasErrors() {
		t.Error("Result should have errors")
	}

	errors := result.Errors()
	if len(errors) != 5 {
		t.Errorf("Expected 5 errors, got %d", len(errors))
	}

	// Verify error details
	for i, err := range errors {
		if err.Index != i {
			t.Errorf("Expected error index %d, got %d", i, err.Index)
		}

		if err.OpID != fmt.Sprintf("op-%d", i) {
			t.Errorf("Expected OpID 'op-%d', got %s", i, err.OpID)
		}
	}
}

// TestContextCancellation tests context cancellation propagation
func TestContextCancellation(t *testing.T) {
	config := config.DefaultConfig()
	ctx := context.NewContext(config)

	// Start a goroutine that waits for cancellation
	done := make(chan bool)
	go func() {
		select {
		case <-ctx.Done():
			done <- true
		case <-time.After(1 * time.Second):
			done <- false
		}
	}()

	// Cancel the context
	ctx.Cancel()

	// Verify cancellation was received
	select {
	case cancelled := <-done:
		if !cancelled {
			t.Error("Context cancellation was not received")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for cancellation")
	}
}

// TestResourceCleanup tests proper resource cleanup
func TestResourceCleanup(t *testing.T) {
	poolManager := pool.NewManager()

	// Get and return many items to test cleanup
	const numItems = 1000

	for i := 0; i < numItems; i++ {
		// Test orchestrator items
		orchItem := poolManager.GetOrchestrator()
		orchItem.Status.Store(uint32(i))
		orchItem.Result = i
		poolManager.PutOrchestrator(orchItem)

		// Test slice items
		slice := poolManager.GetSlice(10)
		slice.Data = append(slice.Data, i)
		poolManager.PutSlice(slice)

		// Test context items
		ctxItem := poolManager.GetContext()
		ctxItem.Values["test"] = i
		poolManager.PutContext(ctxItem)

		// Test result items
		resultItem := poolManager.GetResult()
		resultItem.Entries["test"] = i
		poolManager.PutResult(resultItem)
	}

	// Verify items are properly reset when retrieved again
	orchItem := poolManager.GetOrchestrator()
	if orchItem.Status.Load() != 0 {
		t.Error("Orchestrator item was not properly reset")
	}

	if orchItem.Result != nil {
		t.Error("Orchestrator result was not properly reset")
	}

	slice := poolManager.GetSlice(5)
	if len(slice.Data) != 0 {
		t.Error("Slice was not properly reset")
	}

	ctxItem := poolManager.GetContext()
	if len(ctxItem.Values) != 0 {
		t.Error("Context values were not properly reset")
	}

	resultItem := poolManager.GetResult()
	if len(resultItem.Entries) != 0 {
		t.Error("Result entries were not properly reset")
	}

	// Return items
	poolManager.PutOrchestrator(orchItem)
	poolManager.PutSlice(slice)
	poolManager.PutContext(ctxItem)
	poolManager.PutResult(resultItem)
}

// BenchmarkIntegratedOperations benchmarks all components working together
func BenchmarkIntegratedOperations(b *testing.B) {
	poolManager := pool.NewManager()
	statusManager := status.NewManager()
	config := config.DefaultConfig()
	ctx := context.NewContext(config)
	result := result.NewResult()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Get items from pools
		orchItem := poolManager.GetOrchestrator()
		slice := poolManager.GetSlice(10)
		ctxItem := poolManager.GetContext()
		resultItem := poolManager.GetResult()

		// Perform operations
		statusManager.TransitionToRunning()
		ctx.Set("key", i)
		result.Set("result", i)
		slice.Data = append(slice.Data, i)

		// Return items to pools
		poolManager.PutOrchestrator(orchItem)
		poolManager.PutSlice(slice)
		poolManager.PutContext(ctxItem)
		poolManager.PutResult(resultItem)

		statusManager.Reset()
	}
}

// BenchmarkConcurrentIntegratedOperations benchmarks concurrent integrated operations
func BenchmarkConcurrentIntegratedOperations(b *testing.B) {
	poolManager := pool.NewManager()
	config := config.DefaultConfig()
	ctx := context.NewContext(config)
	result := result.NewResult()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Get items from pools
			orchItem := poolManager.GetOrchestrator()
			slice := poolManager.GetSlice(10)
			ctxItem := poolManager.GetContext()
			resultItem := poolManager.GetResult()

			// Perform operations
			ctx.Get("key")
			result.Get("result")

			// Return items to pools
			poolManager.PutOrchestrator(orchItem)
			poolManager.PutSlice(slice)
			poolManager.PutContext(ctxItem)
			poolManager.PutResult(resultItem)
		}
	})
}
