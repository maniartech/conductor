package atomic

import (
	"fmt"
	"sync"
	"testing"

	. "github.com/maniartech/orchestrator/internal/atomic"
)

// TestCounter tests atomic counter operations
func TestCounter(t *testing.T) {
	t.Run("NewCounter initializes to zero", func(t *testing.T) {
		counter := NewCounter()
		if counter.Load() != 0 {
			t.Errorf("Expected 0, got %d", counter.Load())
		}
	})

	t.Run("NewCounterWithValue initializes correctly", func(t *testing.T) {
		counter := NewCounterWithValue(42)
		if counter.Load() != 42 {
			t.Errorf("Expected 42, got %d", counter.Load())
		}
	})

	t.Run("Add operations work correctly", func(t *testing.T) {
		counter := NewCounter()

		result := counter.Add(5)
		if result != 5 || counter.Load() != 5 {
			t.Errorf("Expected 5, got %d", result)
		}

		result = counter.Add(-2)
		if result != 3 || counter.Load() != 3 {
			t.Errorf("Expected 3, got %d", result)
		}
	})

	t.Run("Inc and Dec operations work correctly", func(t *testing.T) {
		counter := NewCounter()

		result := counter.Inc()
		if result != 1 || counter.Load() != 1 {
			t.Errorf("Expected 1, got %d", result)
		}

		result = counter.Dec()
		if result != 0 || counter.Load() != 0 {
			t.Errorf("Expected 0, got %d", result)
		}
	})

	t.Run("Store and Load operations work correctly", func(t *testing.T) {
		counter := NewCounter()

		counter.Store(100)
		if counter.Load() != 100 {
			t.Errorf("Expected 100, got %d", counter.Load())
		}
	})

	t.Run("CompareAndSwap works correctly", func(t *testing.T) {
		counter := NewCounterWithValue(10)

		// Successful swap
		if !counter.CompareAndSwap(10, 20) {
			t.Error("CompareAndSwap should have succeeded")
		}
		if counter.Load() != 20 {
			t.Errorf("Expected 20, got %d", counter.Load())
		}

		// Failed swap
		if counter.CompareAndSwap(10, 30) {
			t.Error("CompareAndSwap should have failed")
		}
		if counter.Load() != 20 {
			t.Errorf("Expected 20, got %d", counter.Load())
		}
	})

	t.Run("Reset works correctly", func(t *testing.T) {
		counter := NewCounterWithValue(42)

		previous := counter.Reset()
		if previous != 42 {
			t.Errorf("Expected previous value 42, got %d", previous)
		}
		if counter.Load() != 0 {
			t.Errorf("Expected 0 after reset, got %d", counter.Load())
		}
	})
}

// TestFlag tests atomic flag operations
func TestFlag(t *testing.T) {
	t.Run("NewFlag initializes to false", func(t *testing.T) {
		flag := NewFlag()
		if flag.IsSet() {
			t.Error("Expected false, got true")
		}
	})

	t.Run("NewFlagWithValue initializes correctly", func(t *testing.T) {
		flag := NewFlagWithValue(true)
		if !flag.IsSet() {
			t.Error("Expected true, got false")
		}
	})

	t.Run("Set and Clear operations work correctly", func(t *testing.T) {
		flag := NewFlag()

		flag.Set()
		if !flag.IsSet() {
			t.Error("Expected true after Set()")
		}

		flag.Clear()
		if flag.IsSet() {
			t.Error("Expected false after Clear()")
		}
	})

	t.Run("Toggle works correctly", func(t *testing.T) {
		flag := NewFlag()

		result := flag.Toggle()
		if !result || !flag.IsSet() {
			t.Error("Expected true after first toggle")
		}

		result = flag.Toggle()
		if result || flag.IsSet() {
			t.Error("Expected false after second toggle")
		}
	})

	t.Run("SetOnce works correctly", func(t *testing.T) {
		flag := NewFlag()

		// First call should succeed
		if !flag.SetOnce() {
			t.Error("First SetOnce should have succeeded")
		}
		if !flag.IsSet() {
			t.Error("Flag should be set after SetOnce")
		}

		// Second call should fail
		if flag.SetOnce() {
			t.Error("Second SetOnce should have failed")
		}
	})

	t.Run("CompareAndSwap works correctly", func(t *testing.T) {
		flag := NewFlag()

		// Successful swap
		if !flag.CompareAndSwap(false, true) {
			t.Error("CompareAndSwap should have succeeded")
		}
		if !flag.IsSet() {
			t.Error("Flag should be set after successful swap")
		}

		// Failed swap
		if flag.CompareAndSwap(false, false) {
			t.Error("CompareAndSwap should have failed")
		}
		if !flag.IsSet() {
			t.Error("Flag should still be set after failed swap")
		}
	})
}

// TestPointer tests atomic pointer operations
func TestPointer(t *testing.T) {
	t.Run("NewPointer initializes to nil", func(t *testing.T) {
		ptr := NewPointer[int]()
		if ptr.Load() != nil {
			t.Error("Expected nil, got non-nil")
		}
	})

	t.Run("NewPointerWithValue initializes correctly", func(t *testing.T) {
		value := 42
		ptr := NewPointerWithValue(&value)

		loaded := ptr.Load()
		if loaded == nil || *loaded != 42 {
			t.Errorf("Expected pointer to 42, got %v", loaded)
		}
	})

	t.Run("Store and Load operations work correctly", func(t *testing.T) {
		ptr := NewPointer[string]()
		value := "test"

		ptr.Store(&value)
		loaded := ptr.Load()

		if loaded == nil || *loaded != "test" {
			t.Errorf("Expected pointer to 'test', got %v", loaded)
		}
	})

	t.Run("CompareAndSwap works correctly", func(t *testing.T) {
		ptr := NewPointer[int]()
		value1 := 10
		value2 := 20

		ptr.Store(&value1)

		// Successful swap
		if !ptr.CompareAndSwap(&value1, &value2) {
			t.Error("CompareAndSwap should have succeeded")
		}

		loaded := ptr.Load()
		if loaded == nil || *loaded != 20 {
			t.Errorf("Expected pointer to 20, got %v", loaded)
		}

		// Failed swap
		if ptr.CompareAndSwap(&value1, nil) {
			t.Error("CompareAndSwap should have failed")
		}
	})

	t.Run("Swap works correctly", func(t *testing.T) {
		ptr := NewPointer[int]()
		value1 := 10
		value2 := 20

		ptr.Store(&value1)

		previous := ptr.Swap(&value2)
		if previous == nil || *previous != 10 {
			t.Errorf("Expected previous value 10, got %v", previous)
		}

		current := ptr.Load()
		if current == nil || *current != 20 {
			t.Errorf("Expected current value 20, got %v", current)
		}
	})
}

// TestStatus tests atomic status operations
func TestStatus(t *testing.T) {
	t.Run("NewStatus initializes to NotStarted", func(t *testing.T) {
		status := NewStatus()
		if status.Load() != uint32(StatusNotStarted) {
			t.Errorf("Expected %d, got %d", StatusNotStarted, status.Load())
		}
	})

	t.Run("NewStatusWithValue initializes correctly", func(t *testing.T) {
		status := NewStatusWithValue(uint32(StatusRunning))
		if status.Load() != uint32(StatusRunning) {
			t.Errorf("Expected %d, got %d", StatusRunning, status.Load())
		}
	})

	t.Run("IsRunning works correctly", func(t *testing.T) {
		status := NewStatus()

		if status.IsRunning() {
			t.Error("Status should not be running initially")
		}

		status.Store(uint32(StatusRunning))
		if !status.IsRunning() {
			t.Error("Status should be running after setting to Running")
		}
	})

	t.Run("IsCompleted works correctly", func(t *testing.T) {
		status := NewStatus()

		if status.IsCompleted() {
			t.Error("Status should not be completed initially")
		}

		status.Store(uint32(StatusRunning))
		if status.IsCompleted() {
			t.Error("Status should not be completed when running")
		}

		status.Store(uint32(StatusCompleted))
		if !status.IsCompleted() {
			t.Error("Status should be completed when set to Completed")
		}

		status.Store(uint32(StatusCancelled))
		if !status.IsCompleted() {
			t.Error("Status should be completed when set to Cancelled")
		}

		status.Store(uint32(StatusFailed))
		if !status.IsCompleted() {
			t.Error("Status should be completed when set to Failed")
		}
	})
}

// TestMetrics tests atomic metrics collection
func TestMetrics(t *testing.T) {
	t.Run("Counter creation and access", func(t *testing.T) {
		metrics := NewMetrics()

		counter := metrics.Counter("test")
		if counter == nil {
			t.Error("Counter should not be nil")
		}

		// Should return the same counter instance
		counter2 := metrics.Counter("test")
		if counter != counter2 {
			t.Error("Should return the same counter instance")
		}

		counter.Inc()
		if metrics.GetCounterValue("test") != 1 {
			t.Errorf("Expected counter value 1, got %d", metrics.GetCounterValue("test"))
		}
	})

	t.Run("Flag creation and access", func(t *testing.T) {
		metrics := NewMetrics()

		flag := metrics.Flag("test")
		if flag == nil {
			t.Error("Flag should not be nil")
		}

		// Should return the same flag instance
		flag2 := metrics.Flag("test")
		if flag != flag2 {
			t.Error("Should return the same flag instance")
		}

		flag.Set()
		if !metrics.GetFlagValue("test") {
			t.Error("Flag should be set")
		}
	})

	t.Run("Reset clears all metrics", func(t *testing.T) {
		metrics := NewMetrics()

		metrics.Counter("counter1").Add(10)
		metrics.Counter("counter2").Add(20)
		metrics.Flag("flag1").Set()
		metrics.Flag("flag2").Set()

		metrics.Reset()

		if metrics.GetCounterValue("counter1") != 0 {
			t.Error("Counter1 should be reset to 0")
		}
		if metrics.GetCounterValue("counter2") != 0 {
			t.Error("Counter2 should be reset to 0")
		}
		if metrics.GetFlagValue("flag1") {
			t.Error("Flag1 should be reset to false")
		}
		if metrics.GetFlagValue("flag2") {
			t.Error("Flag2 should be reset to false")
		}
	})
}

// TestLockFreeMap tests lock-free map operations
func TestLockFreeMap(t *testing.T) {
	t.Run("Store and Load operations", func(t *testing.T) {
		m := NewLockFreeMap[string, int]()

		m.Store("key1", 10)
		m.Store("key2", 20)

		value, exists := m.Load("key1")
		if !exists || value != 10 {
			t.Errorf("Expected (10, true), got (%d, %t)", value, exists)
		}

		value, exists = m.Load("key2")
		if !exists || value != 20 {
			t.Errorf("Expected (20, true), got (%d, %t)", value, exists)
		}

		_, exists = m.Load("nonexistent")
		if exists {
			t.Error("Expected false for nonexistent key")
		}
	})

	t.Run("Delete operations", func(t *testing.T) {
		m := NewLockFreeMap[string, int]()

		m.Store("key1", 10)
		m.Store("key2", 20)

		m.Delete("key1")

		_, exists := m.Load("key1")
		if exists {
			t.Error("Key1 should be deleted")
		}

		value, exists := m.Load("key2")
		if !exists || value != 20 {
			t.Error("Key2 should still exist")
		}
	})

	t.Run("Len operations", func(t *testing.T) {
		m := NewLockFreeMap[string, int]()

		if m.Len() != 0 {
			t.Errorf("Expected length 0, got %d", m.Len())
		}

		m.Store("key1", 10)
		m.Store("key2", 20)

		if m.Len() != 2 {
			t.Errorf("Expected length 2, got %d", m.Len())
		}

		m.Delete("key1")

		if m.Len() != 1 {
			t.Errorf("Expected length 1, got %d", m.Len())
		}
	})

	t.Run("Range operations", func(t *testing.T) {
		m := NewLockFreeMap[string, int]()

		m.Store("key1", 10)
		m.Store("key2", 20)
		m.Store("key3", 30)

		visited := make(map[string]int)
		m.Range(func(key string, value int) bool {
			visited[key] = value
			return true
		})

		if len(visited) != 3 {
			t.Errorf("Expected 3 visited items, got %d", len(visited))
		}

		for key, expectedValue := range map[string]int{"key1": 10, "key2": 20, "key3": 30} {
			if value, exists := visited[key]; !exists || value != expectedValue {
				t.Errorf("Expected %s=%d, got %d (exists: %t)", key, expectedValue, value, exists)
			}
		}
	})

	t.Run("Range early termination", func(t *testing.T) {
		m := NewLockFreeMap[string, int]()

		m.Store("key1", 10)
		m.Store("key2", 20)
		m.Store("key3", 30)

		count := 0
		m.Range(func(key string, value int) bool {
			count++
			return count < 2 // Stop after 2 items
		})

		if count != 2 {
			t.Errorf("Expected 2 visited items, got %d", count)
		}
	})
}

// TestConcurrentAccess tests thread safety of atomic operations
func TestConcurrentAccess(t *testing.T) {
	t.Run("Concurrent counter operations", func(t *testing.T) {
		counter := NewCounter()
		var wg sync.WaitGroup
		numGoroutines := 100
		numOperations := 1000

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					counter.Inc()
				}
			}()
		}

		wg.Wait()

		expected := int64(numGoroutines * numOperations)
		if counter.Load() != expected {
			t.Errorf("Expected %d, got %d", expected, counter.Load())
		}
	})

	t.Run("Concurrent flag operations", func(t *testing.T) {
		flag := NewFlag()
		var wg sync.WaitGroup
		numGoroutines := 100
		successCount := NewCounter()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if flag.SetOnce() {
					successCount.Inc()
				}
			}()
		}

		wg.Wait()

		if successCount.Load() != 1 {
			t.Errorf("Expected exactly 1 successful SetOnce, got %d", successCount.Load())
		}
		if !flag.IsSet() {
			t.Error("Flag should be set")
		}
	})

	t.Run("Concurrent metrics operations", func(t *testing.T) {
		metrics := NewMetrics()
		var wg sync.WaitGroup
		numGoroutines := 50
		numOperations := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				counterName := fmt.Sprintf("counter_%d", id%10) // 10 different counters
				for j := 0; j < numOperations; j++ {
					metrics.Counter(counterName).Inc()
				}
			}(i)
		}

		wg.Wait()

		// Each counter should have been incremented by 5 goroutines * 100 operations = 500
		for i := 0; i < 10; i++ {
			counterName := fmt.Sprintf("counter_%d", i)
			expected := int64(5 * numOperations) // 5 goroutines per counter
			if metrics.GetCounterValue(counterName) != expected {
				t.Errorf("Counter %s: expected %d, got %d",
					counterName, expected, metrics.GetCounterValue(counterName))
			}
		}
	})
}

// TestMemoryBarriers tests memory barrier functionality
func TestMemoryBarriers(t *testing.T) {
	t.Run("Memory barriers don't panic", func(t *testing.T) {
		var barrier MemoryBarrier

		// These should not panic
		barrier.LoadBarrier()
		barrier.StoreBarrier()
		barrier.FullBarrier()
	})
}

// BenchmarkAtomicOperations benchmarks atomic operations performance
func BenchmarkAtomicOperations(b *testing.B) {
	b.Run("Counter/Inc", func(b *testing.B) {
		counter := NewCounter()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			counter.Inc()
		}
	})

	b.Run("Counter/Load", func(b *testing.B) {
		counter := NewCounterWithValue(42)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = counter.Load()
		}
	})

	b.Run("Flag/IsSet", func(b *testing.B) {
		flag := NewFlagWithValue(true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = flag.IsSet()
		}
	})

	b.Run("Status/IsRunning", func(b *testing.B) {
		status := NewStatusWithValue(uint32(StatusRunning))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = status.IsRunning()
		}
	})

	b.Run("Pointer/Load", func(b *testing.B) {
		value := 42
		ptr := NewPointerWithValue(&value)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ptr.Load()
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent atomic operations
func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("ConcurrentCounter", func(b *testing.B) {
		counter := NewCounter()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				counter.Inc()
			}
		})
	})

	b.Run("ConcurrentFlag", func(b *testing.B) {
		flag := NewFlag()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = flag.IsSet()
			}
		})
	})

	b.Run("ConcurrentMetrics", func(b *testing.B) {
		metrics := NewMetrics()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				metrics.Counter("test").Inc()
			}
		})
	})
}
