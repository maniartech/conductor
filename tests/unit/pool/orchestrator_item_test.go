package pool

import (
	"testing"

	. "github.com/maniartech/orchestrator/internal/pool"
)

func TestOrchestratorItemFields(t *testing.T) {
	item := &OrchestratorItem{}

	// Test initial state
	if item.Status.Load() != 0 {
		t.Error("Status should be 0 initially")
	}

	if item.Result != nil {
		t.Error("Result should be nil initially")
	}

	if item.Error != nil {
		t.Error("Error should be nil initially")
	}

	// Test setting values
	item.Status.Store(1)
	item.Result = "test result"
	item.Error = &testError{}

	// Verify values are set
	if item.Status.Load() != 1 {
		t.Error("Status should be 1")
	}

	if item.Result != "test result" {
		t.Error("Result should be 'test result'")
	}

	if item.Error == nil {
		t.Error("Error should not be nil")
	}
}

func TestOrchestratorItemAtomicOperations(t *testing.T) {
	item := &OrchestratorItem{}

	// Test atomic status operations
	item.Status.Store(42)
	if item.Status.Load() != 42 {
		t.Error("Atomic store/load failed")
	}

	// Test compare and swap
	if !item.Status.CompareAndSwap(42, 100) {
		t.Error("CompareAndSwap should succeed")
	}

	if item.Status.Load() != 100 {
		t.Error("Status should be 100 after CompareAndSwap")
	}

	// Test failed compare and swap
	if item.Status.CompareAndSwap(42, 200) {
		t.Error("CompareAndSwap should fail with wrong old value")
	}

	if item.Status.Load() != 100 {
		t.Error("Status should remain 100 after failed CompareAndSwap")
	}
}

func TestOrchestratorItemConcurrentAccess(t *testing.T) {
	item := &OrchestratorItem{}
	const numGoroutines = 100

	// Test concurrent status updates
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			item.Status.Store(uint32(id))
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Final status should be one of the values (0-99)
	finalStatus := item.Status.Load()
	if finalStatus >= numGoroutines {
		t.Errorf("Final status %d should be less than %d", finalStatus, numGoroutines)
	}
}

// Benchmark tests
func BenchmarkOrchestratorItemFieldAccess(b *testing.B) {
	item := &OrchestratorItem{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item.Status.Store(1)
		item.Result = "test"
		item.Error = &testError{}
		// Reset manually (simulating what manager does)
		item.Status.Store(0)
		item.Result = nil
		item.Error = nil
	}
}

func BenchmarkOrchestratorItemAtomicLoad(b *testing.B) {
	item := &OrchestratorItem{}
	item.Status.Store(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item.Status.Load()
	}
}

func BenchmarkOrchestratorItemAtomicStore(b *testing.B) {
	item := &OrchestratorItem{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item.Status.Store(uint32(i))
	}
}

func BenchmarkOrchestratorItemCompareAndSwap(b *testing.B) {
	item := &OrchestratorItem{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item.Status.Store(0)
		item.Status.CompareAndSwap(0, 1)
	}
}
