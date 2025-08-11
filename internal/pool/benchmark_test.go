package pool

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// BenchmarkOrchestratorPool benchmarks orchestrator pool operations
func BenchmarkOrchestratorPool(b *testing.B) {
	manager := NewManager()

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetOrchestrator()
			manager.PutOrchestrator(item)
		}
	})

	b.Run("GetPut", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetOrchestrator()
			item.Status.Store(uint32(i))
			item.Result = i
			manager.PutOrchestrator(item)
		}
	})

	b.Run("ConcurrentGetPut", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				item := manager.GetOrchestrator()
				item.Status.Store(1)
				manager.PutOrchestrator(item)
			}
		})
	})
}

// BenchmarkSlicePool benchmarks slice pool operations
func BenchmarkSlicePool(b *testing.B) {
	manager := NewManager()

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetSlice(16)
			manager.PutSlice(item)
		}
	})

	b.Run("GetAppendPut", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetSlice(16)
			item.Data = append(item.Data, i)
			manager.PutSlice(item)
		}
	})

	b.Run("ConcurrentGetPut", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				item := manager.GetSlice(16)
				item.Data = append(item.Data, 1, 2, 3)
				manager.PutSlice(item)
			}
		})
	})
}

// BenchmarkContextPool benchmarks context pool operations
func BenchmarkContextPool(b *testing.B) {
	manager := NewManager()

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetContext()
			manager.PutContext(item)
		}
	})

	b.Run("GetSetPut", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetContext()
			item.Values["key"] = i
			manager.PutContext(item)
		}
	})

	b.Run("ConcurrentGetPut", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				item := manager.GetContext()
				item.Values["test"] = "value"
				manager.PutContext(item)
			}
		})
	})
}

// BenchmarkResultPool benchmarks result pool operations
func BenchmarkResultPool(b *testing.B) {
	manager := NewManager()

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetResult()
			manager.PutResult(item)
		}
	})

	b.Run("GetSetPut", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetResult()
			item.Entries["result"] = i
			manager.PutResult(item)
		}
	})

	b.Run("ConcurrentGetPut", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				item := manager.GetResult()
				item.Entries["test"] = "value"
				manager.PutResult(item)
			}
		})
	})
}

// BenchmarkPoolEfficiency compares pooled vs non-pooled allocations
func BenchmarkPoolEfficiency(b *testing.B) {
	manager := NewManager()

	b.Run("WithPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetSlice(100)
			item.Data = append(item.Data, i)
			manager.PutSlice(item)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data := make([]any, 0, 100)
			data = append(data, i)
			_ = data
		}
	})
}

// BenchmarkMemoryEfficiency tests memory allocation efficiency
func BenchmarkMemoryEfficiency(b *testing.B) {
	manager := NewManager()

	// Pre-populate pool
	items := make([]*SliceItem, 100)
	for i := range items {
		items[i] = manager.GetSlice(50)
	}
	for _, item := range items {
		manager.PutSlice(item)
	}

	b.Run("PooledAllocations", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetSlice(50)
			item.Data = append(item.Data, i)
			manager.PutSlice(item)
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Mallocs-m1.Mallocs), "mallocs")
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc), "bytes")
	})

	b.Run("DirectAllocations", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data := make([]any, 0, 50)
			data = append(data, i)
			_ = data
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Mallocs-m1.Mallocs), "mallocs")
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc), "bytes")
	})
}

// BenchmarkStatsTracking benchmarks the overhead of statistics tracking
func BenchmarkStatsTracking(b *testing.B) {
	manager := NewManager()

	b.Run("WithStatsTracking", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetOrchestrator()
			manager.PutOrchestrator(item)
		}
	})

	// Create a manager without stats tracking for comparison
	basicManager := &Manager{
		orchestratorPool: &sync.Pool{
			New: func() any {
				return &OrchestratorItem{
					WG:     sync.WaitGroup{},
					Status: atomic.Uint32{},
				}
			},
		},
	}

	b.Run("WithoutStatsTracking", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := basicManager.orchestratorPool.Get().(*OrchestratorItem)
			item.Status.Store(0)
			item.Result = nil
			item.Error = nil
			basicManager.orchestratorPool.Put(item)
		}
	})
}

// BenchmarkConcurrentStats benchmarks concurrent statistics access
func BenchmarkConcurrentStats(b *testing.B) {
	manager := NewManager()

	// Generate some activity
	for i := 0; i < 1000; i++ {
		item := manager.GetOrchestrator()
		manager.PutOrchestrator(item)
	}

	b.Run("StatsAccess", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = manager.Stats()
			}
		})
	})

	b.Run("ConcurrentStatsAndOperations", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if pb.Next() {
					_ = manager.Stats()
				} else {
					item := manager.GetOrchestrator()
					manager.PutOrchestrator(item)
				}
			}
		})
	})
}

// BenchmarkLeakPrevention benchmarks the overhead of leak prevention mechanisms
func BenchmarkLeakPrevention(b *testing.B) {
	manager := NewManager()

	b.Run("SliceWithLeakPrevention", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetSlice(16)
			// Simulate some data
			for j := 0; j < 10; j++ {
				item.Data = append(item.Data, j)
			}
			manager.PutSlice(item) // This will reset the slice
		}
	})

	b.Run("ContextWithLeakPrevention", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := manager.GetContext()
			// Simulate some data
			item.Values["key1"] = "value1"
			item.Values["key2"] = "value2"
			item.Values["key3"] = "value3"
			manager.PutContext(item) // This will clear the map
		}
	})
}

// BenchmarkDefaultManager benchmarks the global default manager
func BenchmarkDefaultManager(b *testing.B) {
	b.Run("DefaultManagerOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := DefaultManager.GetOrchestrator()
			DefaultManager.PutOrchestrator(item)
		}
	})

	b.Run("ConcurrentDefaultManager", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				item := DefaultManager.GetSlice(16)
				item.Data = append(item.Data, 1)
				DefaultManager.PutSlice(item)
			}
		})
	})
}
