// Package pool provides zero-allocation object pooling system using sync.Pool
// with proper lifecycle management for the orchestrator library.
package pool

// PoolStats contains comprehensive pool usage metrics for monitoring and optimization
type PoolStats struct {
	// Orchestrator pool metrics
	OrchestratorGets   int64
	OrchestratorPuts   int64
	OrchestratorNews   int64 // New objects created
	OrchestratorReuses int64 // Objects reused from pool

	// Slice pool metrics
	SliceGets   int64
	SlicePuts   int64
	SliceNews   int64
	SliceReuses int64

	// Context pool metrics
	ContextGets   int64
	ContextPuts   int64
	ContextNews   int64
	ContextReuses int64

	// Result pool metrics
	ResultGets   int64
	ResultPuts   int64
	ResultNews   int64
	ResultReuses int64

	// Efficiency metrics
	TotalGets        int64
	TotalPuts        int64
	TotalNews        int64
	TotalReuses      int64
	OverallReuseRate float64 // Percentage of gets that were reuses
}

// CalculateEfficiency computes efficiency metrics from raw counters
func (ps *PoolStats) CalculateEfficiency() {
	ps.TotalGets = ps.OrchestratorGets + ps.SliceGets + ps.ContextGets + ps.ResultGets
	ps.TotalPuts = ps.OrchestratorPuts + ps.SlicePuts + ps.ContextPuts + ps.ResultPuts
	ps.TotalNews = ps.OrchestratorNews + ps.SliceNews + ps.ContextNews + ps.ResultNews
	ps.TotalReuses = ps.OrchestratorReuses + ps.SliceReuses + ps.ContextReuses + ps.ResultReuses

	if ps.TotalGets > 0 {
		ps.OverallReuseRate = float64(ps.TotalReuses) / float64(ps.TotalGets) * 100
	}
}
