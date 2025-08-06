// Package context provides context management for orchestration operations.
// It includes the Context interface and implementation for storing values,
// configuration, and handling cancellation across orchestration boundaries.
package context

import (
	"context"
	"sync"

	"github.com/maniartech/orchestrator/internal/config"
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
	Config() config.Config

	// Cancel signals cancellation to the orchestration
	Cancel()

	// Done returns a channel that's closed when cancellation occurs
	Done() <-chan struct{}
}

// contextImpl provides a concrete implementation of Context
type contextImpl struct {
	values map[string]any
	config config.Config
	cancel context.CancelFunc
	done   <-chan struct{}
	mu     sync.RWMutex
}

// NewContext creates a new context implementation with the given configuration.
// The context inherits from DefaultConfig() and applies local overrides.
//
// Example:
//
//	cfg := config.Config{Timeout: 30*time.Second}
//	ctx := NewContext(cfg)
//	ctx.Set("start_time", time.Now())
func NewContext(cfg config.Config) Context {
	ctx, cancel := context.WithCancel(cfg.Context)

	return &contextImpl{
		values: make(map[string]any),
		config: cfg.Inherit(config.DefaultConfig()),
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
func (c *contextImpl) Config() config.Config {
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
