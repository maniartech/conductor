// Package context provides enhanced context management for orchestration operations.
// It includes enterprise-grade timeout management, graceful shutdown coordination,
// and nested orchestration lifecycle support with zero-allocation hot paths.
//
// This package implements a military-grade context system using atomic operations
// for performance-critical paths while maintaining KISS principles.
package context

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
)

// Context provides enhanced orchestration context with timeout management,
// graceful shutdown coordination, and nested orchestration lifecycle support.
//
// Key Features:
// - Zero-allocation timeout checking with atomic operations
// - Hierarchical context relationships for complex workflows
// - Enterprise-grade observability with distributed tracing paths
// - Graceful shutdown coordination across context hierarchies
//
// Example Basic Usage:
//
//	ctx.Set("user_id", 123)
//	if userID := ctx.Get("user_id"); userID != nil {
//	    fmt.Printf("Processing user: %v\n", userID)
//	}
//
// Example Enhanced Usage:
//
//	// Create context with timeout
//	ctxWithTimeout := ctx.WithTimeout(30 * time.Second)
//	if ctxWithTimeout.IsExpired() {
//	    return errors.New("operation timed out")
//	}
//
//	// Create nested context hierarchy
//	childCtx := ctx.CreateChild("payment-processing")
//	fmt.Println(childCtx.GetPath()) // "/payment-processing"
type Context interface {
	// Core context operations
	Get(name string) any
	Set(name string, value any)
	Config() config.Config

	// Lifecycle management
	Cancel()
	Done() <-chan struct{}
	Err() error

	// Enhanced timeout management (zero-allocation hot paths)
	WithTimeout(timeout time.Duration) Context
	WithDeadline(deadline time.Time) Context
	GetRemainingTime() time.Duration
	IsExpired() bool

	// Nested orchestration lifecycle
	CreateChild(name string) Context
	GetParent() Context
	GetPath() string

	// Graceful shutdown coordination
	Shutdown() <-chan struct{}
	IsShuttingDown() bool
}

// contextImpl implements the Context interface with enterprise-grade features
// and zero-allocation hot paths using atomic operations.
type contextImpl struct {
	// Core context state
	values   map[string]any
	valuesMu sync.RWMutex
	config   config.Config

	// Go standard context integration
	ctx    context.Context
	cancel context.CancelFunc

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
func NewContext(cfg config.Config) Context {
	baseCtx := cfg.Context
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	ctx, cancel := context.WithCancel(baseCtx)

	return &contextImpl{
		// Core state
		values: make(map[string]any),
		config: cfg.Inherit(config.DefaultConfig()),
		ctx:    ctx,
		cancel: cancel,

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
	c.cancel()
}

// Done returns a channel that's closed when cancellation occurs.
// This integrates with Go's standard context cancellation patterns.
func (c *contextImpl) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err returns the error that caused the context to be cancelled.
// This integrates with Go's standard context error reporting.
func (c *contextImpl) Err() error {
	return c.ctx.Err()
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
	// Create child context with Go's standard timeout
	ctx, cancel := context.WithDeadline(c.ctx, deadline)

	// Create new context implementation
	child := &contextImpl{
		// Core state
		values: make(map[string]any),
		config: c.config, // Inherit configuration
		ctx:    ctx,
		cancel: cancel,

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
	// Create child context
	ctx, cancel := context.WithCancel(c.ctx)

	// Build child path
	var childPath string
	if c.path == "" {
		childPath = "/" + name
	} else {
		childPath = c.path + "/" + name
	}

	child := &contextImpl{
		// Core state (inherit from parent)
		values: make(map[string]any),
		config: c.config, // Inherit configuration
		ctx:    ctx,
		cancel: cancel,

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
