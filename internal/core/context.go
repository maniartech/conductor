// Package core provides the fundamental types and interfaces for the
// orchestrator library following Go best practices and KISS principles.
package core

import (
	"context"
	"sync"
)

// Context provides access to orchestration state and configuration.
// It enables inter-task communication and provides access to orchestration configuration.
//
// Example:
//
//	ctx.Set("user_id", 123)
//	if userID := ctx.Get("user_id"); userID != nil {
//	    fmt.Printf("Processing user: %v\n", userID)
//	}
type Context interface {
	// Get retrieves a value by name from the context
	Get(name string) any

	// Set stores a value by name in the context
	Set(name string, value any)

	// Config returns the current configuration
	Config() Config

	// Cancel signals cancellation to the orchestration
	Cancel()

	// Done returns a channel that's closed when cancellation occurs
	Done() <-chan struct{}
}

// contextImpl provides a concrete implementation of Context
type contextImpl struct {
	values map[string]any
	config Config
	cancel context.CancelFunc
	done   <-chan struct{}
	mu     sync.RWMutex
}

// NewContext creates a new context implementation with the given configuration.
// The context inherits from DefaultConfig() and applies local overrides.
//
// Example:
//
//	config := Config{Timeout: 30*time.Second}
//	ctx := NewContext(config)
//	ctx.Set("start_time", time.Now())
func NewContext(config Config) Context {
	ctx, cancel := context.WithCancel(config.Context)

	return &contextImpl{
		values: make(map[string]any),
		config: config.Inherit(DefaultConfig()),
		cancel: cancel,
		done:   ctx.Done(),
	}
}

// Get retrieves a value by name from the context
func (c *contextImpl) Get(name string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.values[name]
}

// Set stores a value by name in the context
func (c *contextImpl) Set(name string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[name] = value
}

// Config returns the current configuration
func (c *contextImpl) Config() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// Cancel signals cancellation to the orchestration
func (c *contextImpl) Cancel() {
	c.cancel()
}

// Done returns a channel that's closed when cancellation occurs
func (c *contextImpl) Done() <-chan struct{} {
	return c.done
}
