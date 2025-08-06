// Package pool provides zero-allocation object pooling system using sync.Pool
// with proper lifecycle management for the orchestrator library.
package pool

// ContextItem represents a pooled context component
type ContextItem struct {
	Values map[string]any
	Done   chan struct{}
}
