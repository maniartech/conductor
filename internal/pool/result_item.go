// Package pool provides zero-allocation object pooling system using sync.Pool
// with proper lifecycle management for the orchestrator library.
package pool

import (
	"sync"
)

// ResultItem represents a pooled result container
type ResultItem struct {
	Entries map[string]any
	Errors  []error
	Mutex   sync.RWMutex
}
