// Package pool provides zero-allocation object pooling system using sync.Pool
// with proper lifecycle management for the orchestrator library.
package pool

import (
	"sync"
	"sync/atomic"
)

// Manager handles object pooling for reusable orchestrator components
type Manager struct {
	orchestratorPool *sync.Pool
	slicePool        *sync.Pool
	contextPool      *sync.Pool
	resultPool       *sync.Pool

	// Metrics for monitoring pool usage
	orchestratorGets atomic.Int64
	orchestratorPuts atomic.Int64
	sliceGets        atomic.Int64
	slicePuts        atomic.Int64
}

// NewManager creates a new pool manager with proper initialization
func NewManager() *Manager {
	m := &Manager{
		orchestratorPool: &sync.Pool{
			New: func() any {
				return &OrchestratorItem{
					WG:     sync.WaitGroup{},
					Status: atomic.Uint32{},
				}
			},
		},
		slicePool: &sync.Pool{
			New: func() any {
				return &SliceItem{
					Data: make([]any, 0, 16),
					Cap:  16,
				}
			},
		},
		contextPool: &sync.Pool{
			New: func() any {
				return &ContextItem{
					Values: make(map[string]any, 8),
					Done:   make(chan struct{}),
				}
			},
		},
		resultPool: &sync.Pool{
			New: func() any {
				return &ResultItem{
					Entries: make(map[string]any, 8),
					Errors:  make([]error, 0, 4),
					Mutex:   sync.RWMutex{},
				}
			},
		},
	}
	return m
}

// GetOrchestrator retrieves an orchestrator item from the pool
func (m *Manager) GetOrchestrator() *OrchestratorItem {
	m.orchestratorGets.Add(1)
	item := m.orchestratorPool.Get().(*OrchestratorItem)
	// Reset the item to clean state
	item.Status.Store(0)
	item.Result = nil
	item.Error = nil
	return item
}

// PutOrchestrator returns an orchestrator item to the pool
func (m *Manager) PutOrchestrator(item *OrchestratorItem) {
	if item == nil {
		return
	}
	m.orchestratorPuts.Add(1)
	m.orchestratorPool.Put(item)
}

// GetSlice retrieves a slice item from the pool
func (m *Manager) GetSlice(minCap int) *SliceItem {
	m.sliceGets.Add(1)
	item := m.slicePool.Get().(*SliceItem)

	// Ensure capacity is sufficient
	if cap(item.Data) < minCap {
		item.Data = make([]any, 0, minCap)
		item.Cap = minCap
	} else {
		// Reset slice to zero length but keep capacity
		item.Data = item.Data[:0]
	}

	return item
}

// PutSlice returns a slice item to the pool
func (m *Manager) PutSlice(item *SliceItem) {
	if item == nil {
		return
	}
	m.slicePuts.Add(1)
	// Only pool slices that aren't too large to prevent memory bloat
	if cap(item.Data) <= 1024 {
		m.slicePool.Put(item)
	}
}

// GetContext retrieves a context item from the pool
func (m *Manager) GetContext() *ContextItem {
	item := m.contextPool.Get().(*ContextItem)
	// Clear the values map
	for k := range item.Values {
		delete(item.Values, k)
	}
	// Ensure done channel is fresh
	select {
	case <-item.Done:
		item.Done = make(chan struct{})
	default:
		// Channel is already fresh
	}
	return item
}

// PutContext returns a context item to the pool
func (m *Manager) PutContext(item *ContextItem) {
	if item == nil {
		return
	}
	m.contextPool.Put(item)
}

// GetResult retrieves a result item from the pool
func (m *Manager) GetResult() *ResultItem {
	item := m.resultPool.Get().(*ResultItem)
	// Clear the entries map
	for k := range item.Entries {
		delete(item.Entries, k)
	}
	// Reset errors slice
	item.Errors = item.Errors[:0]
	return item
}

// PutResult returns a result item to the pool
func (m *Manager) PutResult(item *ResultItem) {
	if item == nil {
		return
	}
	m.resultPool.Put(item)
}

// Stats returns pool usage statistics
func (m *Manager) Stats() PoolStats {
	return PoolStats{
		OrchestratorGets: m.orchestratorGets.Load(),
		OrchestratorPuts: m.orchestratorPuts.Load(),
		SliceGets:        m.sliceGets.Load(),
		SlicePuts:        m.slicePuts.Load(),
	}
}

// Reset clears all pool statistics
func (m *Manager) Reset() {
	m.orchestratorGets.Store(0)
	m.orchestratorPuts.Store(0)
	m.sliceGets.Store(0)
	m.slicePuts.Store(0)
}

// DefaultManager is the global pool manager instance
var DefaultManager = NewManager()
