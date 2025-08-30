package context

import (
	"context"
	"sync/atomic"

	"github.com/maniartech/orchestrator/pkg/config"
)

// NewContext creates a new enhanced context with the provided configuration.
// The context provides thread-safe access to shared state, configuration,
// and enterprise-grade lifecycle management.
//
// Example:
//
//	cfg := config.DefaultConfig()
//	ctx := NewContext(cfg)
//	ctx.Set("start_time", time.Now())
//
//	// Enhanced timeout management
//	ctxWithTimeout := ctx.WithTimeout(30 * time.Second)
//
//	// Nested orchestration
//	childCtx := ctx.CreateChild("database-operations")
func newContextImpl(cfg config.Config) *contextImpl {
	baseCtx := cfg.Context
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	// Create cause-aware cancelable context (Go 1.20+)
	ctx, cancel := context.WithCancelCause(baseCtx)

	return &contextImpl{
		// Core state
		values:      make(map[string]any),
		config:      cfg.Inherit(config.DefaultConfig()),
		ctx:         ctx,
		cancelCause: cancel,

		// Timeout management
		deadline:     atomic.Int64{},
		timeoutSet:   atomic.Bool{},
		timeoutTimer: nil,

		// Hierarchy
		parent:   nil,
		children: make([]*contextImpl, 0, 4), // Pre-allocate capacity
		name:     "root",
		path:     "",
		depth:    0,

		// Shutdown coordination
		shutdownInitiated: atomic.Bool{},
		shutdownChan:      make(chan struct{}), // Pre-allocated
	}
}

// NewContext returns a consumer-facing Context. Cancellation is available for
// backwards compatibility via the Context interface, but new code should prefer
// NewContextWithController to obtain an owner-only Controller for cancellation.
func NewContext(cfg config.Config) Context {
	return newContextImpl(cfg)
}

// NewContextWithController returns a Context and its owner-only Controller.
// Use the returned Controller to cancel with or without a cause.
func NewContextWithController(cfg config.Config) (Context, Controller) {
	impl := newContextImpl(cfg)
	return impl, impl
}
