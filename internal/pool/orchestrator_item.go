// Package pool provides zero-allocation object pooling system using sync.Pool
// with proper lifecycle management for the orchestrator library.
package pool

import (
	"sync"
	"sync/atomic"
)

// OrchestratorItem represents a pooled orchestrator component
type OrchestratorItem struct {
	WG     sync.WaitGroup
	Status atomic.Uint32
	Result any
	Error  error
}
