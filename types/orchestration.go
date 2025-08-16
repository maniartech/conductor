// Package types provides core interfaces and types for the orchestrator library.
// This package contains all the fundamental interfaces that can be implemented
// by library users to extend the orchestrator with custom orchestration types.
package types

import (
	"context"

	"github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/result"
	"github.com/maniartech/orchestrator/pkg/config"
)

// Orchestration defines the interface for all orchestration types.
// It provides a fluent API for configuring orchestration behavior following the builder pattern.
// All orchestration types (Task, Sequential, Concurrent, Conditional) implement this interface.
//
// Example:
//
//	task := Task(myFunction).
//	    Named("my-task").
//	    With(Config{Timeout: 30*time.Second}).
//	    ErrorBoundary(CollectAll)
type Orchestration interface {
	// Named sets a name for the orchestration for observability and debugging.
	// The name appears in logs and error messages to help identify which orchestration failed.
	//
	// Parameters:
	//   - name: Human-readable name for the orchestration
	//
	// Returns:
	//   - Orchestration: The same orchestration instance for method chaining
	//
	// Example:
	//
	//	task := Task(fetchUser).Named("fetch-user-data")
	Named(name string) Orchestration

	// With applies configuration to the orchestration.
	// Configuration is inherited hierarchically with local overrides.
	//
	// Parameters:
	//   - config: Configuration to apply to this orchestration
	//
	// Returns:
	//   - Orchestration: The same orchestration instance for method chaining
	//
	// Example:
	//
	//	task := Task(longRunningTask).With(Config{Timeout: 60*time.Second})
	With(config config.Config) Orchestration

	// ErrorBoundary sets the error handling strategy for this orchestration.
	// This determines how errors are handled and propagated.
	//
	// Parameters:
	//   - strategy: Error handling strategy (FailFast or CollectAll)
	//
	// Returns:
	//   - Orchestration: The same orchestration instance for method chaining
	//
	// Example:
	//
	//	task := Task(riskyOperation).ErrorBoundary(CollectAll)
	ErrorBoundary(strategy errors.ErrorStrategy) Orchestration

	// Execute runs the orchestration with the provided context and configuration.
	// This method is thread-safe and can be called multiple times.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - config: Runtime configuration that overrides orchestration-level config
	//
	// Returns:
	//   - *result.Result: Execution results with named outputs
	//   - error: Execution error if the orchestration failed
	//
	// Example:
	//
	//	result, err := orchestration.Execute(ctx, config.Config{})
	//	if err != nil {
	//	    log.Printf("Orchestration failed: %v", err)
	//	}
	Execute(ctx context.Context, config config.Config) (*result.Result, error)

	// GetName returns the orchestration name for debugging and observability.
	// Returns empty string if no name was set.
	GetName() string

	// GetConfig returns the orchestration's configuration.
	// Returns nil if no configuration was set.
	GetConfig() *config.Config

	// GetStatus returns the current orchestration status.
	// This method is thread-safe and uses atomic operations.
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
