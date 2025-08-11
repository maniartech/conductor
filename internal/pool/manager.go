// Package pool provides zero-allocation object pooling system using sync.Pool
// with proper lifecycle management for the orchestrator library.
//
// This enhanced pool system provides:
// - Comprehensive metrics tracking for all pool types
// - Resource leak prevention mechanisms
// - Efficient slice allocation and reuse patterns
// - Memory pool management for temporary objects
package pool

import (
	"sync"
	"sync/atomic"
	"time"
)

// Manager handles object pooling for reusable orchestrator components with
// enterprise-grade metrics tracking and leak prevention mechanisms.
type Manager struct {
	orchestratorPool *sync.Pool
	slicePool        *sync.Pool
	contextPool      *sync.Pool
	resultPool       *sync.Pool

	// Enhanced metrics for comprehensive monitoring
	orchestratorGets   atomic.Int64
	orchestratorPuts   atomic.Int64
	orchestratorNews   atomic.Int64
	orchestratorReuses atomic.Int64

	sliceGets   atomic.Int64
	slicePuts   atomic.Int64
	sliceNews   atomic.Int64
	sliceReuses atomic.Int64

	contextGets   atomic.Int64
	contextPuts   atomic.Int64
	contextNews   atomic.Int64
	contextReuses atomic.Int64

	resultGets   atomic.Int64
	resultPuts   atomic.Int64
	resultNews   atomic.Int64
	resultReuses atomic.Int64

	// Leak prevention tracking
	startTime        time.Time
	lastStatsReset   time.Time
	maxSliceCapacity int // Maximum allowed slice capacity before discarding
}

// NewManager creates a new pool manager with proper initialization and
// enhanced metrics tracking for enterprise-grade resource management.
func NewManager() *Manager {
	now := time.Now()
	m := &Manager{
		// Initialize leak prevention tracking
		startTime:        now,
		lastStatsReset:   now,
		maxSliceCapacity: 1024, // Prevent memory bloat from oversized slices
	}

	// Initialize pools with proper closures for metrics tracking
	m.orchestratorPool = &sync.Pool{
		New: func() any {
			// Track new object creation
			m.orchestratorNews.Add(1)
			return &OrchestratorItem{
				WG:     sync.WaitGroup{},
				Status: atomic.Uint32{},
			}
		},
	}

	m.slicePool = &sync.Pool{
		New: func() any {
			// Track new object creation
			m.sliceNews.Add(1)
			return &SliceItem{
				Data: make([]any, 0, 16),
				Cap:  16,
			}
		},
	}

	m.contextPool = &sync.Pool{
		New: func() any {
			// Track new object creation
			m.contextNews.Add(1)
			return &ContextItem{
				Values: make(map[string]any, 8),
				Done:   make(chan struct{}),
			}
		},
	}

	m.resultPool = &sync.Pool{
		New: func() any {
			// Track new object creation
			m.resultNews.Add(1)
			return &ResultItem{
				Entries: make(map[string]any, 8),
				Errors:  make([]error, 0, 4),
				Mutex:   sync.RWMutex{},
			}
		},
	}

	return m
}

// GetOrchestrator retrieves an orchestrator item from the pool with
// comprehensive metrics tracking and proper resource cleanup.
func (m *Manager) GetOrchestrator() *OrchestratorItem {
	gets := m.orchestratorGets.Add(1)
	news := m.orchestratorNews.Load()

	// Track reuse if this get is not creating a new object
	if gets > news {
		m.orchestratorReuses.Add(1)
	}

	item := m.orchestratorPool.Get().(*OrchestratorItem)

	// Reset the item to clean state (leak prevention)
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

// GetSlice retrieves a slice item from the pool with efficient capacity management
// and leak prevention mechanisms.
func (m *Manager) GetSlice(minCap int) *SliceItem {
	gets := m.sliceGets.Add(1)
	news := m.sliceNews.Load()

	// Track reuse if this get is not creating a new object
	if gets > news {
		m.sliceReuses.Add(1)
	}

	item := m.slicePool.Get().(*SliceItem)

	// Ensure capacity is sufficient
	if cap(item.Data) < minCap {
		item.Data = make([]any, 0, minCap)
		item.Cap = minCap
	} else {
		// Reset slice to zero length but keep capacity (efficient reuse)
		item.Data = item.Data[:0]
	}

	return item
}

// PutSlice returns a slice item to the pool with enhanced leak prevention.
// Oversized slices are discarded to prevent memory bloat.
func (m *Manager) PutSlice(item *SliceItem) {
	if item == nil {
		return
	}

	m.slicePuts.Add(1)

	// Enhanced leak prevention: only pool slices within reasonable size limits
	if cap(item.Data) <= m.maxSliceCapacity {
		// Reset slice to prevent data leaks
		item.Data = item.Data[:0]
		m.slicePool.Put(item)
	}
	// Oversized slices are discarded (not pooled) to prevent memory bloat
}

// GetContext retrieves a context item from the pool with comprehensive
// cleanup and metrics tracking.
func (m *Manager) GetContext() *ContextItem {
	gets := m.contextGets.Add(1)
	news := m.contextNews.Load()

	// Track reuse if this get is not creating a new object
	if gets > news {
		m.contextReuses.Add(1)
	}

	item := m.contextPool.Get().(*ContextItem)

	// Comprehensive cleanup to prevent data leaks
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

// PutContext returns a context item to the pool with proper cleanup.
func (m *Manager) PutContext(item *ContextItem) {
	if item == nil {
		return
	}

	m.contextPuts.Add(1)

	// Additional cleanup to prevent resource leaks
	for k := range item.Values {
		delete(item.Values, k)
	}

	m.contextPool.Put(item)
}

// GetResult retrieves a result item from the pool with comprehensive
// cleanup and metrics tracking.
func (m *Manager) GetResult() *ResultItem {
	gets := m.resultGets.Add(1)
	news := m.resultNews.Load()

	// Track reuse if this get is not creating a new object
	if gets > news {
		m.resultReuses.Add(1)
	}

	item := m.resultPool.Get().(*ResultItem)

	// Comprehensive cleanup to prevent data leaks
	for k := range item.Entries {
		delete(item.Entries, k)
	}

	// Reset errors slice but keep capacity for efficiency
	item.Errors = item.Errors[:0]

	return item
}

// PutResult returns a result item to the pool with proper cleanup.
func (m *Manager) PutResult(item *ResultItem) {
	if item == nil {
		return
	}

	m.resultPuts.Add(1)

	// Additional cleanup to prevent resource leaks
	for k := range item.Entries {
		delete(item.Entries, k)
	}
	item.Errors = item.Errors[:0]

	m.resultPool.Put(item)
}

// Stats returns comprehensive pool usage statistics for monitoring and optimization.
func (m *Manager) Stats() PoolStats {
	stats := PoolStats{
		// Orchestrator pool metrics
		OrchestratorGets:   m.orchestratorGets.Load(),
		OrchestratorPuts:   m.orchestratorPuts.Load(),
		OrchestratorNews:   m.orchestratorNews.Load(),
		OrchestratorReuses: m.orchestratorReuses.Load(),

		// Slice pool metrics
		SliceGets:   m.sliceGets.Load(),
		SlicePuts:   m.slicePuts.Load(),
		SliceNews:   m.sliceNews.Load(),
		SliceReuses: m.sliceReuses.Load(),

		// Context pool metrics
		ContextGets:   m.contextGets.Load(),
		ContextPuts:   m.contextPuts.Load(),
		ContextNews:   m.contextNews.Load(),
		ContextReuses: m.contextReuses.Load(),

		// Result pool metrics
		ResultGets:   m.resultGets.Load(),
		ResultPuts:   m.resultPuts.Load(),
		ResultNews:   m.resultNews.Load(),
		ResultReuses: m.resultReuses.Load(),
	}

	// Calculate efficiency metrics
	stats.CalculateEfficiency()

	return stats
}

// Reset clears all pool statistics and updates the reset timestamp.
func (m *Manager) Reset() {
	// Reset orchestrator pool metrics
	m.orchestratorGets.Store(0)
	m.orchestratorPuts.Store(0)
	m.orchestratorNews.Store(0)
	m.orchestratorReuses.Store(0)

	// Reset slice pool metrics
	m.sliceGets.Store(0)
	m.slicePuts.Store(0)
	m.sliceNews.Store(0)
	m.sliceReuses.Store(0)

	// Reset context pool metrics
	m.contextGets.Store(0)
	m.contextPuts.Store(0)
	m.contextNews.Store(0)
	m.contextReuses.Store(0)

	// Reset result pool metrics
	m.resultGets.Store(0)
	m.resultPuts.Store(0)
	m.resultNews.Store(0)
	m.resultReuses.Store(0)

	// Update reset timestamp
	m.lastStatsReset = time.Now()
}

// GetUptime returns the time since the manager was created.
func (m *Manager) GetUptime() time.Duration {
	return time.Since(m.startTime)
}

// GetTimeSinceReset returns the time since statistics were last reset.
func (m *Manager) GetTimeSinceReset() time.Duration {
	return time.Since(m.lastStatsReset)
}

// SetMaxSliceCapacity sets the maximum allowed slice capacity before discarding.
// This helps prevent memory bloat from oversized slices.
func (m *Manager) SetMaxSliceCapacity(maxCap int) {
	m.maxSliceCapacity = maxCap
}

// GetMaxSliceCapacity returns the current maximum slice capacity limit.
func (m *Manager) GetMaxSliceCapacity() int {
	return m.maxSliceCapacity
}

// DefaultManager is the global pool manager instance
var DefaultManager = NewManager()
