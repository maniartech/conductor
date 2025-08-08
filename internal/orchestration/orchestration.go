// Package orchestration provides the core interfaces and types for orchestration operations.
// It defines the Orchestration interface that all orchestration components must implement,
// enabling fluent API design and hierarchical configuration management.
package orchestration

import (
	"context"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/result"
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
	With(config config.Config) Orchestration

	// ErrorBoundary sets error handling strategy for this orchestration.
	// Controls how errors propagate within this orchestration scope.
	ErrorBoundary(strategy errors.ErrorStrategy) Orchestration

	// Execute runs the orchestration and returns the result
	Execute(ctx context.Context, config config.Config) (*result.Result, error)

	// GetName returns the orchestration name for debugging and observability.
	// Returns empty string if no name was set.
	GetName() string

	// GetConfig returns the orchestration's configuration.
	// Returns nil if no configuration was set.
	GetConfig() *config.Config

	// GetStatus returns the current orchestration status using atomic operations.
	// This method is thread-safe and can be called concurrently.
	GetStatus() Status

	// PathResolver methods - all orchestrations support path-based lookup
	PathResolver
}

// Executor defines the interface for executing orchestrations
// This is kept for backward compatibility and future extensions
type Executor interface {
	// Execute runs the orchestration and returns the result
	Execute(ctx context.Context, config config.Config) (*result.Result, error)
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
