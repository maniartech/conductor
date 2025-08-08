// Package orchestration provides the core interfaces and types for orchestration operations.
// It defines the Orchestration interface that all orchestration components must implement,
// enabling fluent API design and hierarchical configuration management.
package orchestration

import "github.com/maniartech/orchestrator/types"

// Orchestration is an alias to the types.Orchestration interface for backward compatibility.
// All orchestration implementations should use types.Orchestration directly.
type Orchestration = types.Orchestration

// Executor is an alias to the types.Executor interface for backward compatibility.
// All executor implementations should use types.Executor directly.
type Executor = types.Executor

// OrchestrationFunc is a function type that implements Orchestration.
// It allows simple functions to be used as orchestrations.
//
// Example:
//
//	fn := OrchestrationFunc(func() (any, error) {
//	    return "Hello, World!", nil
//	})
type OrchestrationFunc func() (any, error)
