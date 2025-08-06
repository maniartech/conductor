// Package core provides the fundamental types and interfaces for the
// orchestrator library following Go best practices and KISS principles.
package core

import (
	"context"
)

// Orchestration defines the interface for all orchestration types.
// It provides a fluent API for configuring orchestration behavior following the builder pattern.
// All orchestration types (Task, Sequential, Concurrent, Conditional) implement this interface.
//
// Example:
//
//	orchestration := someOrchestration.
//	    Named("user-processing").
//	    With(Config{Timeout: 30*time.Second}).
//	    ErrorBoundary(CollectAll)
type Orchestration interface {
	// Named sets a name for the orchestration for observability and debugging.
	// The name appears in logs and error messages to help identify which orchestration failed.
	Named(name string) Orchestration

	// With applies configuration to the orchestration.
	// Configuration is inherited hierarchically with local overrides.
	With(config Config) Orchestration

	// ErrorBoundary sets error handling strategy for this orchestration.
	// Controls how errors propagate within this orchestration scope.
	ErrorBoundary(strategy ErrorStrategy) Orchestration

	// Execute runs the orchestration and returns the result
	Execute(ctx context.Context, config Config) (*Result, error)
}

// Executor defines the interface for executing orchestrations
// This is kept for backward compatibility and future extensions
type Executor interface {
	// Execute runs the orchestration and returns the result
	Execute(ctx context.Context, config Config) (*Result, error)
}

// OrchestrationFunc is a function type that implements Orchestration.
// It allows simple functions to be used as orchestrations.
//
// Example:
//
//	fn := OrchestrationFunc(func() (any, error) {
//	    return "Hello, World!", nil
//	})
type OrchestrationFunc func() (any, error)
