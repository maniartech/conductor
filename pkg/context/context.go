// Package context provides enhanced context management for orchestration operations.
// It includes enterprise-grade timeout management, graceful shutdown coordination,
// and nested orchestration lifecycle support with zero-allocation hot paths.
//
// This package implements a military-grade context system using atomic operations
// for performance-critical paths while maintaining KISS principles.
package context

import (
	"time"

	"github.com/maniartech/orchestrator/pkg/config"
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
	// Satisfy context.Context
	Deadline() (time.Time, bool)
	// Cancel cancels this context. Deprecated for consumer code; owners should use Controller.
	// Deprecated: Prefer obtaining a Controller and invoking Cancel/CancelWithReason there.
	Cancel()
	Done() <-chan struct{}
	Err() error
	Value(key any) any

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
