// Package core provides the fundamental types and interfaces for the
// orchestrator library following Go best practices and KISS principles.
package core

import (
	"sync"
)

// Result contains named outputs and errors from orchestration execution.
// It provides thread-safe access to orchestration results with type safety.
//
// Example:
//
//	result := NewResult()
//	result.Set("user", User{ID: 123, Name: "John"})
//	if user, ok := result.GetTyped[User]("user"); ok {
//	    fmt.Printf("User: %+v\n", user)
//	}
type Result struct {
	entries map[string]any
	errors  []OperationError
	mu      sync.RWMutex
}

// NewResult creates a new result container with initialized maps and slices.
//
// Example:
//
//	result := NewResult()
//	result.Set("status", "completed")
func NewResult() *Result {
	return &Result{
		entries: make(map[string]any),
		errors:  make([]OperationError, 0),
	}
}

// Get retrieves a value by name from the result.
// Returns nil if the key doesn't exist.
//
// Example:
//
//	value := result.Get("status")
//	if value != nil {
//	    fmt.Printf("Status: %v\n", value)
//	}
func (r *Result) Get(name string) any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entries[name]
}

// GetTyped retrieves a value by name with type safety using generics.
// Returns the typed value and a boolean indicating if the value exists and matches the type.
//
// Example:
//
//	result.Set("count", 42)
//	if value, ok := GetTyped[int](result, "count"); ok {
//	    fmt.Printf("Count: %d\n", value)
//	}
func GetTyped[T any](r *Result, name string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var zero T
	value, exists := r.entries[name]
	if !exists {
		return zero, false
	}

	if typed, ok := value.(T); ok {
		return typed, true
	}

	return zero, false
}

// Set stores a value by name in the result.
// Thread-safe operation that can be called concurrently.
//
// Example:
//
//	result.Set("user_count", 42)
//	result.Set("processing_time", time.Since(start))
func (r *Result) Set(name string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[name] = value
}

// HasErrors returns true if there are any errors in the result
func (r *Result) HasErrors() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.errors) > 0
}

// Errors returns a copy of all errors
func (r *Result) Errors() []OperationError {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.errors) == 0 {
		return nil
	}

	// Return a copy to prevent external modification
	errors := make([]OperationError, len(r.errors))
	copy(errors, r.errors)
	return errors
}

// AddError adds an error to the result
func (r *Result) AddError(err OperationError) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, err)
}
