package pool

import (
	"sync"
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.orchestratorPool == nil {
		t.Error("orchestratorPool is nil")
	}

	if manager.slicePool == nil {
		t.Error("slicePool is nil")
	}

	if manager.contextPool == nil {
		t.Error("contextPool is nil")
	}

	if manager.resultPool == nil {
		t.Error("resultPool is nil")
	}
}

func TestOrchestratorPoolOperations(t *testing.T) {
	manager := NewManager()

	// Test Get operation
	item := manager.GetOrchestrator()
	if item == nil {
		t.Fatal("GetOrchestrator() returned nil")
	}

	// Verify initial state
	if item.Status.Load() != 0 {
		t.Error("Expected status to be 0, got", item.Status.Load())
	}

	if item.Result != nil {
		t.Error("Expected result to be nil")
	}

	if item.Error != nil {
		t.Error("Expected error to be nil")
	}

	// Modify item
	item.Status.Store(1)
	item.Result = "test"
	item.Error = &testError{}

	// Test Put operation
	manager.PutOrchestrator(item)

	// Get another item and verify it's reset
	item2 := manager.GetOrchestrator()
	if item2.Status.Load() != 0 {
		t.Error("Expected status to be reset to 0")
	}

	if item2.Result != nil {
		t.Error("Expected result to be reset to nil")
	}

	if item2.Error != nil {
		t.Error("Expected error to be reset to nil")
	}
}

func TestSlicePoolOperations(t *testing.T) {
	manager := NewManager()

	// Test Get with minimum capacity
	item := manager.GetSlice(10)
	if item == nil {
		t.Fatal("GetSlice() returned nil")
	}

	if len(item.Data) != 0 {
		t.Error("Expected slice length to be 0, got", len(item.Data))
	}

	if cap(item.Data) < 10 {
		t.Error("Expected slice capacity to be at least 10, got", cap(item.Data))
	}

	// Test with larger capacity requirement
	item2 := manager.GetSlice(50)
	if cap(item2.Data) < 50 {
		t.Error("Expected slice capacity to be at least 50, got", cap(item2.Data))
	}

	// Add some data
	item.Data = append(item.Data, "test1", "test2")

	// Put back and get again
	manager.PutSlice(item)
	item3 := manager.GetSlice(5)

	// Should be reset to zero length
	if len(item3.Data) != 0 {
		t.Error("Expected slice to be reset to zero length")
	}
}

func TestContextPoolOperations(t *testing.T) {
	manager := NewManager()

	item := manager.GetContext()
	if item == nil {
		t.Fatal("GetContext() returned nil")
	}

	if item.Values == nil {
		t.Error("Expected Values map to be initialized")
	}

	if len(item.Values) != 0 {
		t.Error("Expected Values map to be empty")
	}

	if item.Done == nil {
		t.Error("Expected Done channel to be initialized")
	}

	// Add some values
	item.Values["test"] = "value"

	// Put back and get again
	manager.PutContext(item)
	item2 := manager.GetContext()

	// Should be reset
	if len(item2.Values) != 0 {
		t.Error("Expected Values map to be cleared")
	}
}

func TestResultPoolOperations(t *testing.T) {
	manager := NewManager()

	item := manager.GetResult()
	if item == nil {
		t.Fatal("GetResult() returned nil")
	}

	if item.Entries == nil {
		t.Error("Expected Entries map to be initialized")
	}

	if len(item.Entries) != 0 {
		t.Error("Expected Entries map to be empty")
	}

	if item.Errors == nil {
		t.Error("Expected Errors slice to be initialized")
	}

	if len(item.Errors) != 0 {
		t.Error("Expected Errors slice to be empty")
	}

	// Add some data
	item.Entries["test"] = "value"
	item.Errors = append(item.Errors, &testError{})

	// Put back and get again
	manager.PutResult(item)
	item2 := manager.GetResult()

	// Should be reset
	if len(item2.Entries) != 0 {
		t.Error("Expected Entries map to be cleared")
	}

	if len(item2.Errors) != 0 {
		t.Error("Expected Errors slice to be cleared")
	}
}

func TestPoolStats(t *testing.T) {
	manager := NewManager()

	// Initial stats should be zero
	stats := manager.Stats()
	if stats.OrchestratorGets != 0 {
		t.Error("Expected OrchestratorGets to be 0")
	}

	if stats.OrchestratorPuts != 0 {
		t.Error("Expected OrchestratorPuts to be 0")
	}

	// Perform some operations
	item := manager.GetOrchestrator()
	manager.PutOrchestrator(item)

	slice := manager.GetSlice(10)
	manager.PutSlice(slice)

	// Check updated stats
	stats = manager.Stats()
	if stats.OrchestratorGets != 1 {
		t.Error("Expected OrchestratorGets to be 1, got", stats.OrchestratorGets)
	}

	if stats.OrchestratorPuts != 1 {
		t.Error("Expected OrchestratorPuts to be 1, got", stats.OrchestratorPuts)
	}

	if stats.SliceGets != 1 {
		t.Error("Expected SliceGets to be 1, got", stats.SliceGets)
	}

	if stats.SlicePuts != 1 {
		t.Error("Expected SlicePuts to be 1, got", stats.SlicePuts)
	}
}

func TestPoolReset(t *testing.T) {
	manager := NewManager()

	// Perform some operations
	manager.GetOrchestrator()
	manager.GetSlice(10)

	// Reset stats
	manager.Reset()

	stats := manager.Stats()
	if stats.OrchestratorGets != 0 {
		t.Error("Expected OrchestratorGets to be 0 after reset")
	}

	if stats.SliceGets != 0 {
		t.Error("Expected SliceGets to be 0 after reset")
	}
}

func TestConcurrentPoolAccess(t *testing.T) {
	manager := NewManager()
	const numGoroutines = 100
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Test orchestrator pool
				item := manager.GetOrchestrator()
				item.Status.Store(uint32(j))
				manager.PutOrchestrator(item)

				// Test slice pool
				slice := manager.GetSlice(10)
				slice.Data = append(slice.Data, j)
				manager.PutSlice(slice)

				// Test context pool
				ctx := manager.GetContext()
				ctx.Values["test"] = j
				manager.PutContext(ctx)

				// Test result pool
				result := manager.GetResult()
				result.Entries["test"] = j
				manager.PutResult(result)
			}
		}()
	}

	wg.Wait()

	// Verify stats
	stats := manager.Stats()
	expectedOps := int64(numGoroutines * operationsPerGoroutine)

	if stats.OrchestratorGets != expectedOps {
		t.Errorf("Expected %d orchestrator gets, got %d", expectedOps, stats.OrchestratorGets)
	}

	if stats.OrchestratorPuts != expectedOps {
		t.Errorf("Expected %d orchestrator puts, got %d", expectedOps, stats.OrchestratorPuts)
	}
}

func TestNilHandling(t *testing.T) {
	manager := NewManager()

	// Test putting nil items doesn't panic
	manager.PutOrchestrator(nil)
	manager.PutSlice(nil)
	manager.PutContext(nil)
	manager.PutResult(nil)
}

func TestSliceCapacityLimit(t *testing.T) {
	manager := NewManager()

	// Create a slice with large capacity
	slice := manager.GetSlice(2000)
	slice.Data = make([]interface{}, 0, 2000)

	// Put it back - should not be pooled due to size limit
	manager.PutSlice(slice)

	// Get a new slice - should be a fresh one, not the large one
	newSlice := manager.GetSlice(10)
	if cap(newSlice.Data) >= 2000 {
		t.Error("Large slice should not have been pooled")
	}
}

func TestDefaultManager(t *testing.T) {
	if DefaultManager == nil {
		t.Fatal("DefaultManager is nil")
	}

	// Test that default manager works
	item := DefaultManager.GetOrchestrator()
	if item == nil {
		t.Error("DefaultManager.GetOrchestrator() returned nil")
	}

	DefaultManager.PutOrchestrator(item)
}

// Benchmark tests
func BenchmarkOrchestratorPoolGet(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := manager.GetOrchestrator()
		manager.PutOrchestrator(item)
	}
}

func BenchmarkSlicePoolGet(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := manager.GetSlice(16)
		manager.PutSlice(item)
	}
}

func BenchmarkConcurrentPoolAccess(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			item := manager.GetOrchestrator()
			manager.PutOrchestrator(item)
		}
	})
}

// Test helper
type testError struct{}

func (e *testError) Error() string {
	return "test error"
}
