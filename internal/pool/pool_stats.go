// Package pool provides zero-allocation object pooling system using sync.Pool
// with proper lifecycle management for the orchestrator library.
package pool

// PoolStats contains pool usage metrics
type PoolStats struct {
	OrchestratorGets int64
	OrchestratorPuts int64
	SliceGets        int64
	SlicePuts        int64
}
