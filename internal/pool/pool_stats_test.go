package pool

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestPoolStatsInitialization(t *testing.T) {
	stats := &PoolStats{}

	// All fields should be zero initially
	if stats.OrchestratorGets != 0 {
		t.Error("OrchestratorGets should be 0 initially")
	}

	if stats.OrchestratorPuts != 0 {
		t.Error("OrchestratorPuts should be 0 initially")
	}

	if stats.SliceGets != 0 {
		t.Error("SliceGets should be 0 initially")
	}

	if stats.SlicePuts != 0 {
		t.Error("SlicePuts should be 0 initially")
	}

}

func TestPoolStatsAtomicOperations(t *testing.T) {
	stats := &PoolStats{}

	// Test atomic increments
	stats.OrchestratorGets++
	stats.OrchestratorPuts++
	stats.SliceGets++
	stats.SlicePuts++

	// Verify increments
	if stats.OrchestratorGets != 1 {
		t.Error("OrchestratorGets should be 1")
	}

	if stats.OrchestratorPuts != 1 {
		t.Error("OrchestratorPuts should be 1")
	}

	if stats.SliceGets != 1 {
		t.Error("SliceGets should be 1")
	}

	if stats.SlicePuts != 1 {
		t.Error("SlicePuts should be 1")
	}

}

func TestPoolStatsConcurrentAccess(t *testing.T) {
	// Note: PoolStats fields are not atomic - the Manager handles atomicity
	// This test demonstrates that direct concurrent access to PoolStats is not safe
	// In practice, stats should only be accessed through the Manager
	stats := &PoolStats{}
	const numGoroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 4) // 4 different stat types

	// Concurrent increments for each stat type
	for i := 0; i < numGoroutines; i++ {
		// OrchestratorGets
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				atomic.AddInt64(&stats.OrchestratorGets, 1)
			}
		}()

		// OrchestratorPuts
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				atomic.AddInt64(&stats.OrchestratorPuts, 1)
			}
		}()

		// SliceGets
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				atomic.AddInt64(&stats.SliceGets, 1)
			}
		}()

		// SlicePuts
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				atomic.AddInt64(&stats.SlicePuts, 1)
			}
		}()
	}

	wg.Wait()

	// Verify that increments happened using atomic reads
	if atomic.LoadInt64(&stats.OrchestratorGets) == 0 {
		t.Error("OrchestratorGets should be greater than 0")
	}

	if atomic.LoadInt64(&stats.OrchestratorPuts) == 0 {
		t.Error("OrchestratorPuts should be greater than 0")
	}

	if atomic.LoadInt64(&stats.SliceGets) == 0 {
		t.Error("SliceGets should be greater than 0")
	}

	if atomic.LoadInt64(&stats.SlicePuts) == 0 {
		t.Error("SlicePuts should be greater than 0")
	}
}

func TestPoolStatsReset(t *testing.T) {
	stats := &PoolStats{
		OrchestratorGets: 100,
		OrchestratorPuts: 90,
		SliceGets:        200,
		SlicePuts:        180,
	}

	// Reset all stats
	*stats = PoolStats{}

	// Verify all are zero
	if stats.OrchestratorGets != 0 {
		t.Error("OrchestratorGets should be 0 after reset")
	}

	if stats.OrchestratorPuts != 0 {
		t.Error("OrchestratorPuts should be 0 after reset")
	}

	if stats.SliceGets != 0 {
		t.Error("SliceGets should be 0 after reset")
	}

	if stats.SlicePuts != 0 {
		t.Error("SlicePuts should be 0 after reset")
	}
}

func TestPoolStatsBalanceTracking(t *testing.T) {
	stats := &PoolStats{}

	// Simulate balanced operations
	for i := 0; i < 100; i++ {
		stats.OrchestratorGets++
		stats.OrchestratorPuts++
		stats.SliceGets++
		stats.SlicePuts++
	}

	// All gets and puts should be equal
	if stats.OrchestratorGets != stats.OrchestratorPuts {
		t.Error("OrchestratorGets and OrchestratorPuts should be equal")
	}

	if stats.SliceGets != stats.SlicePuts {
		t.Error("SliceGets and SlicePuts should be equal")
	}
}

func TestPoolStatsImbalanceDetection(t *testing.T) {
	stats := &PoolStats{}

	// Simulate imbalanced operations (more gets than puts)
	for i := 0; i < 100; i++ {
		stats.OrchestratorGets++
		if i%2 == 0 { // Only put back half
			stats.OrchestratorPuts++
		}
	}

	// Should detect imbalance
	if stats.OrchestratorGets <= stats.OrchestratorPuts {
		t.Error("Should have more gets than puts")
	}

	expectedGets := int64(100)
	expectedPuts := int64(50)

	if stats.OrchestratorGets != expectedGets {
		t.Errorf("Expected %d gets, got %d", expectedGets, stats.OrchestratorGets)
	}

	if stats.OrchestratorPuts != expectedPuts {
		t.Errorf("Expected %d puts, got %d", expectedPuts, stats.OrchestratorPuts)
	}
}

// Benchmark tests
func BenchmarkPoolStatsIncrement(b *testing.B) {
	stats := &PoolStats{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats.OrchestratorGets++
	}
}

func BenchmarkPoolStatsConcurrentIncrement(b *testing.B) {
	stats := &PoolStats{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			stats.OrchestratorGets++
		}
	})
}

func BenchmarkPoolStatsMultipleFields(b *testing.B) {
	stats := &PoolStats{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats.OrchestratorGets++
		stats.OrchestratorPuts++
		stats.SliceGets++
		stats.SlicePuts++
	}
}

func BenchmarkPoolStatsRead(b *testing.B) {
	stats := &PoolStats{
		OrchestratorGets: 1000,
		OrchestratorPuts: 900,
		SliceGets:        2000,
		SlicePuts:        1800,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stats.OrchestratorGets
		_ = stats.OrchestratorPuts
		_ = stats.SliceGets
		_ = stats.SlicePuts
	}
}
