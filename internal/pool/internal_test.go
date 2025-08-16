package pool

import (
	"sync"
	"sync/atomic"
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
