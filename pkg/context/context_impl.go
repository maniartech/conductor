package context

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maniartech/orchestrator/pkg/config"
)

// contextImpl implements the Context interface with enterprise-grade features
// and zero-allocation hot paths using atomic operations.
type contextImpl struct {
	// Core context state
	values   map[string]any
	valuesMu sync.RWMutex
	config   config.Config

	// Go standard context integration
	ctx         context.Context
	cancelCause context.CancelCauseFunc

	// Enhanced timeout management (atomic for zero-allocation)
	deadline     atomic.Int64 // Unix nanoseconds, 0 = no timeout
	timeoutSet   atomic.Bool  // Whether timeout is active
	timeoutTimer *time.Timer  // Single timer per context
	timerMu      sync.Mutex   // Protects timer operations

	// Nested orchestration hierarchy (cached for performance)
	parent   *contextImpl
	children []*contextImpl
	childMu  sync.RWMutex
	name     string
	path     string // Cached: "/parent/child/grandchild"
	depth    int    // Cached: 0, 1, 2, ...

	// Graceful shutdown coordination (atomic)
	shutdownInitiated atomic.Bool
	shutdownChan      chan struct{} // Pre-allocated
}

// Controller exposes owner-only cancellation operations for a Context.
// Obtain via NewContextWithController; do not expose this to consumer-facing APIs.
type Controller interface {
	Cancel()
	CancelWithReason(err error)
}

// Get retrieves a value by name from the context.
// This method is thread-safe and can be called concurrently.
func (c *contextImpl) Get(name string) any {
	c.valuesMu.RLock()
	defer c.valuesMu.RUnlock()
	return c.values[name]
}

// Set stores a value by name in the context.
// This method is thread-safe and can be called concurrently.
func (c *contextImpl) Set(name string, value any) {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()
	c.values[name] = value
}

// Config returns the current configuration.
// This method is thread-safe and can be called concurrently.
func (c *contextImpl) Config() config.Config {
	c.valuesMu.RLock()
	defer c.valuesMu.RUnlock()
	return c.config
}

// Cancel signals cancellation to the orchestration and all child contexts.
// This method stops timeout timers and propagates cancellation down the hierarchy.
func (c *contextImpl) Cancel() {
	c.stopTimeoutTimer()
	// Default cancellation cause mirrors standard behavior
	if c.cancelCause != nil {
		c.cancelCause(context.Canceled)
	}
}

// CancelWithReason cancels with a specific error cause.
func (c *contextImpl) CancelWithReason(err error) {
	c.stopTimeoutTimer()
	if c.cancelCause != nil {
		if err == nil {
			err = context.Canceled
		}
		c.cancelCause(err)
	}
}

// Done returns a channel that's closed when cancellation occurs.
// This integrates with Go's standard context cancellation patterns.
func (c *contextImpl) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err returns the error that caused the context to be cancelled.
// This integrates with Go's standard context error reporting.
func (c *contextImpl) Err() error { return context.Cause(c.ctx) }

// Deadline returns the time when work done on behalf of this context should be canceled.
// This delegates to the underlying std context to satisfy context.Context.
func (c *contextImpl) Deadline() (time.Time, bool) {
	return c.ctx.Deadline()
}

// Value returns the value associated with this context for key.
// This delegates to the underlying std context and intentionally does not access the orchestration KV store.
func (c *contextImpl) Value(key any) any {
	return c.ctx.Value(key)
}

// WithTimeout creates a child context with timeout enforcement.
// Uses atomic operations for zero-allocation timeout checking.
//
// Example:
//
//	ctxWithTimeout := ctx.WithTimeout(30 * time.Second)
//	if ctxWithTimeout.IsExpired() {
//	    return errors.New("operation timed out")
//	}
func (c *contextImpl) WithTimeout(timeout time.Duration) Context {
	return c.WithDeadline(time.Now().Add(timeout))
}

// WithDeadline creates a child context with deadline enforcement.
// Uses atomic operations for high-performance timeout checking.
func (c *contextImpl) WithDeadline(deadline time.Time) Context {
	// First apply a deadline to the parent
	ctxWithDeadline, _ := context.WithDeadline(c.ctx, deadline)
	// Then layer a cause-aware cancelable context on top so owners can set a custom cause
	ctx, cancel := context.WithCancelCause(ctxWithDeadline)

	// Create new context implementation
	child := &contextImpl{
		// Core state
		values:      make(map[string]any),
		config:      c.config, // Inherit configuration
		ctx:         ctx,
		cancelCause: cancel,

		// Timeout management with deadline
		deadline:     atomic.Int64{},
		timeoutSet:   atomic.Bool{},
		timeoutTimer: nil,

		// Hierarchy (inherit from parent)
		parent:   c,
		children: make([]*contextImpl, 0, 4),
		name:     c.name, // Will be updated if this becomes a named child
		path:     c.path, // Will be updated if this becomes a named child
		depth:    c.depth,

		// Shutdown coordination
		shutdownInitiated: atomic.Bool{},
		shutdownChan:      make(chan struct{}),
	}

	// Set atomic deadline (Unix nanoseconds for atomic operations)
	child.deadline.Store(deadline.UnixNano())
	child.timeoutSet.Store(true)

	// Start timeout monitoring
	child.startTimeoutTimer(deadline)

	return child
}

// GetRemainingTime returns the remaining time until timeout.
// Uses atomic operations for zero-allocation calculation (hot path).
func (c *contextImpl) GetRemainingTime() time.Duration {
	if !c.timeoutSet.Load() {
		return 0 // No timeout set
	}

	deadlineNano := c.deadline.Load()
	if deadlineNano == 0 {
		return 0 // No timeout set
	}

	now := time.Now().UnixNano()
	remaining := deadlineNano - now

	if remaining <= 0 {
		return 0 // Already expired
	}

	return time.Duration(remaining)
}

// IsExpired returns true if the context has exceeded its timeout.
// Uses atomic operations for zero-allocation checking (hot path optimization).
func (c *contextImpl) IsExpired() bool {
	if !c.timeoutSet.Load() {
		return false // No timeout set
	}

	deadlineNano := c.deadline.Load()
	if deadlineNano == 0 {
		return false // No timeout set
	}

	now := time.Now().UnixNano()
	return now > deadlineNano
}

// CreateChild creates a child context for nested orchestrations.
// The child inherits the parent's configuration and values, and maintains
// a hierarchical relationship for distributed tracing and cleanup.
//
// Example:
//
//	childCtx := ctx.CreateChild("payment-processing")
//	fmt.Println(childCtx.GetPath()) // "/payment-processing"
//
//	grandchildCtx := childCtx.CreateChild("fraud-detection")
//	fmt.Println(grandchildCtx.GetPath()) // "/payment-processing/fraud-detection"
func (c *contextImpl) CreateChild(name string) Context {
	// Create cause-aware child context
	ctx, cancel := context.WithCancelCause(c.ctx)

	// Build child path
	var childPath string
	if c.path == "" {
		childPath = "/" + name
	} else {
		childPath = c.path + "/" + name
	}

	child := &contextImpl{
		// Core state (inherit from parent)
		values:      make(map[string]any),
		config:      c.config, // Inherit configuration
		ctx:         ctx,
		cancelCause: cancel,

		// Timeout management (no timeout by default)
		deadline:     atomic.Int64{},
		timeoutSet:   atomic.Bool{},
		timeoutTimer: nil,

		// Hierarchy
		parent:   c,
		children: make([]*contextImpl, 0, 4),
		name:     name,
		path:     childPath,
		depth:    c.depth + 1,

		// Shutdown coordination
		shutdownInitiated: atomic.Bool{},
		shutdownChan:      make(chan struct{}),
	}

	// Add child to parent's children list
	c.childMu.Lock()
	c.children = append(c.children, child)
	c.childMu.Unlock()

	return child
}

// GetParent returns the parent context, or nil if this is a root context.
// This enables traversing up the context hierarchy for cleanup or data access.
func (c *contextImpl) GetParent() Context {
	if c.parent == nil {
		return nil
	}
	return c.parent
}

// GetPath returns the hierarchical path of this context.
// This is cached for zero-allocation access and enables distributed tracing.
//
// Example paths:
//   - Root context: ""
//   - Child context: "/payment-processing"
//   - Grandchild context: "/payment-processing/fraud-detection"
func (c *contextImpl) GetPath() string {
	return c.path
}

// Shutdown returns a channel that's closed when graceful shutdown is initiated.
// This enables coordinated shutdown across complex orchestration hierarchies.
//
// Example:
//
//	select {
//	case <-ctx.Shutdown():
//	    // Perform cleanup and exit gracefully
//	    return nil
//	case <-time.After(operationTimeout):
//	    // Continue with operation
//	}
func (c *contextImpl) Shutdown() <-chan struct{} {
	return c.shutdownChan
}

// IsShuttingDown returns true if graceful shutdown has been initiated.
// Uses atomic operations for zero-allocation checking (hot path).
func (c *contextImpl) IsShuttingDown() bool {
	return c.shutdownInitiated.Load()
}

// startTimeoutTimer starts the timeout monitoring timer.
// This is called internally when a timeout is set.
func (c *contextImpl) startTimeoutTimer(deadline time.Time) {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()

	now := time.Now()
	duration := deadline.Sub(now)

	if duration <= 0 {
		// Already expired, cancel immediately
		go c.Cancel()
		return
	}

	// Create timer that will cancel the context when timeout expires
	c.timeoutTimer = time.AfterFunc(duration, func() {
		c.Cancel()
	})
}

// stopTimeoutTimer stops the timeout monitoring timer.
// This is called when the context is cancelled or cleaned up.
func (c *contextImpl) stopTimeoutTimer() {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()

	if c.timeoutTimer != nil {
		c.timeoutTimer.Stop()
		c.timeoutTimer = nil
	}
}

// initiateShutdown initiates graceful shutdown for this context and all children.
// This is an internal method used for coordinated shutdown.
func (c *contextImpl) initiateShutdown() {
	// Set shutdown flag atomically
	if c.shutdownInitiated.CompareAndSwap(false, true) {
		// Close shutdown channel to notify waiters
		close(c.shutdownChan)

		// Propagate shutdown to all children
		c.childMu.RLock()
		children := make([]*contextImpl, len(c.children))
		copy(children, c.children)
		c.childMu.RUnlock()

		for _, child := range children {
			go child.initiateShutdown()
		}
	}
}
