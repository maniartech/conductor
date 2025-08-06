// Package pool provides zero-allocation object pooling system using sync.Pool
// with proper lifecycle management for the orchestrator library.
package pool

// SliceItem represents a pooled slice for results
type SliceItem struct {
	Data []any
	Cap  int
}
