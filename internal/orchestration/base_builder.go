// Package orchestration provides common abstractions and base implementations
// for all orchestration types in the orchestrator library.
package orchestration

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/types"
)

// BaseOrchestrationBuilder provides common functionality for all orchestration builders.
// This eliminates code duplication and ensures consistent behavior across all orchestration types.
//
// All orchestration builders should embed this struct to inherit:
//   - Fluent API methods (Named, With, ErrorBoundary)
//   - Atomic status management
//   - Configuration inheritance
//   - Operation ID generation
//   - Common validation logic
type BaseOrchestrationBuilder struct {
	name              string
	config            *config.Config
	errorBoundary     *errors.ErrorStrategy
	status            atomic.Uint32
	orchestrationType string // "task", "sequential", "concurrent", etc.
}

// NewBaseOrchestrationBuilder creates a new base orchestration builder.
// This should be called by all orchestration constructors.
//
// Parameters:
//   - orchestrationType: The type of orchestration ("task", "sequential", "concurrent", etc.)
//
// Returns:
//   - *BaseOrchestrationBuilder: A new base builder instance
func NewBaseOrchestrationBuilder(orchestrationType string) *BaseOrchestrationBuilder {
	return &BaseOrchestrationBuilder{
		orchestrationType: orchestrationType,
	}
}

// Named sets a name for the orchestration for observability and debugging.
// The name appears in logs and error messages to help identify which orchestration failed.
// This method should be called by the concrete orchestration's Named method.
//
// Example:
//
//	func (sb *SequentialBuilder) Named(name string) types.Orchestration {
//	    sb.BaseOrchestrationBuilder.SetName(name)
//	    return sb
//	}
func (bob *BaseOrchestrationBuilder) SetName(name string) {
	bob.name = name
}

// With applies configuration to the orchestration.
// Configuration is inherited hierarchically with local overrides.
// This method should be called by the concrete orchestration's With method.
//
// Example:
//
//	func (sb *SequentialBuilder) With(config config.Config) types.Orchestration {
//	    bob.BaseOrchestrationBuilder.SetConfig(config)
//	    return sb
//	}
func (bob *BaseOrchestrationBuilder) SetConfig(config config.Config) {
	bob.config = &config
}

// ErrorBoundary sets error handling strategy for this orchestration.
// This controls how errors propagate within this orchestration scope.
// This method should be called by the concrete orchestration's ErrorBoundary method.
//
// Example:
//
//	func (sb *SequentialBuilder) ErrorBoundary(strategy errors.ErrorStrategy) types.Orchestration {
//	    bob.BaseOrchestrationBuilder.SetErrorBoundary(strategy)
//	    return sb
//	}
func (bob *BaseOrchestrationBuilder) SetErrorBoundary(strategy errors.ErrorStrategy) {
	bob.errorBoundary = &strategy
}

// GetName returns the orchestration name for debugging and observability.
// Returns empty string if no name was set.
func (bob *BaseOrchestrationBuilder) GetName() string {
	return bob.name
}

// GetType returns the orchestration type as a string.
// This is useful for logging and debugging to identify the orchestration type.
func (bob *BaseOrchestrationBuilder) GetType() string {
	return bob.orchestrationType
}

// GetConfig returns the orchestration's configuration.
// Returns nil if no configuration was set.
func (bob *BaseOrchestrationBuilder) GetConfig() *config.Config {
	return bob.config
}

// GetErrorBoundary returns the orchestration's error boundary strategy.
// Returns nil if no error boundary was set.
func (bob *BaseOrchestrationBuilder) GetErrorBoundary() *errors.ErrorStrategy {
	return bob.errorBoundary
}

// GetStatus returns the current orchestration status using atomic operations.
// This method is thread-safe and can be called concurrently.
func (bob *BaseOrchestrationBuilder) GetStatus() types.Status {
	return types.Status(bob.status.Load())
}

// SetStatus atomically sets the orchestration status.
// This is an internal method used during orchestration execution.
func (bob *BaseOrchestrationBuilder) SetStatus(status types.Status) {
	bob.status.Store(uint32(status))
}

// CompareAndSwapStatus atomically compares and swaps the orchestration status.
// Returns true if the swap was successful, false otherwise.
// This ensures thread-safe status transitions.
func (bob *BaseOrchestrationBuilder) CompareAndSwapStatus(old, new types.Status) bool {
	return bob.status.CompareAndSwap(uint32(old), uint32(new))
}

// GetOperationID generates a unique operation ID for traceability.
// Uses the orchestration name if available, otherwise generates a default ID.
//
// Parameters:
//   - instance: Pointer to the concrete orchestration instance for fallback ID generation
//
// Returns:
//   - string: Unique operation ID
func (bob *BaseOrchestrationBuilder) GetOperationID(instance any) string {
	if bob.name != "" {
		return fmt.Sprintf("%s-%s", bob.orchestrationType, bob.name)
	}
	return fmt.Sprintf("%s-%p", bob.orchestrationType, instance)
}

// ApplyConfigurationInheritance applies configuration inheritance with local overrides.
// This is a common pattern used by all orchestrations during execution.
//
// Parameters:
//   - parentConfig: Configuration from parent orchestration or execution context
//
// Returns:
//   - config.Config: Final configuration with inheritance applied
func (bob *BaseOrchestrationBuilder) ApplyConfigurationInheritance(parentConfig config.Config) config.Config {
	finalConfig := parentConfig
	if bob.config != nil {
		finalConfig = bob.config.Inherit(parentConfig)
	}

	// Apply error boundary if specified
	if bob.errorBoundary != nil {
		finalConfig.ErrorStrategy = *bob.errorBoundary
	}

	return finalConfig
}

// ValidateExecutionPreconditions checks common preconditions before execution.
// This prevents double execution and ensures orchestration is in valid state.
//
// Returns:
//   - error: Error if preconditions are not met
func (bob *BaseOrchestrationBuilder) ValidateExecutionPreconditions() error {
	if !bob.CompareAndSwapStatus(types.NotStarted, types.Running) {
		return fmt.Errorf("%s orchestration already executed or in progress, current status: %v",
			bob.orchestrationType, bob.GetStatus())
	}
	return nil
}

// CompleteExecution updates the orchestration status based on execution outcome.
// This should be called at the end of every orchestration's Execute method.
//
// Parameters:
//   - ctx: Execution context to check for cancellation
//   - err: Execution error (nil if successful)
func (bob *BaseOrchestrationBuilder) CompleteExecution(ctx context.Context, err error) {
	if err != nil {
		if ctx.Err() != nil {
			bob.SetStatus(types.Cancelled)
		} else {
			bob.SetStatus(types.Completed)
		}
	} else {
		bob.SetStatus(types.Completed)
	}
}

// =============================================================================
// Common Path Resolution Methods
// =============================================================================

// GetCurrentPath returns the current orchestration's path.
// For leaf orchestrations (like tasks), this is just the operation ID.
// For container orchestrations, this should be overridden to provide hierarchical paths.
func (bob *BaseOrchestrationBuilder) GetCurrentPath(instance any) string {
	return bob.GetOperationID(instance)
}

// GetByPath finds an orchestration by its hierarchical path.
// For leaf orchestrations, this only matches if the path equals the current path.
// Container orchestrations should override this to search their children.
func (bob *BaseOrchestrationBuilder) GetByPath(instance any, path string) (types.Orchestration, error) {
	currentPath := bob.GetCurrentPath(instance)
	if path == currentPath {
		// This is a bit tricky - we need to return the concrete orchestration instance
		// The instance parameter should be the concrete orchestration (e.g., *TaskBuilder)
		if orch, ok := instance.(types.Orchestration); ok {
			return orch, nil
		}
		return nil, fmt.Errorf("instance is not an orchestration: %T", instance)
	}
	return nil, fmt.Errorf("path not found: %s", path)
}

// ListAllPaths returns all available paths in the orchestration subtree.
// For leaf orchestrations, this is just the current path.
// Container orchestrations should override this to include children paths.
func (bob *BaseOrchestrationBuilder) ListAllPaths(instance any) []string {
	return []string{bob.GetCurrentPath(instance)}
}

// FindByName searches for orchestrations by name.
// For leaf orchestrations, this returns the orchestration itself if the name matches.
// Container orchestrations should override this to search children.
func (bob *BaseOrchestrationBuilder) FindByName(instance any, name string) []types.PathMatch {
	if bob.GetName() == name {
		if orch, ok := instance.(types.Orchestration); ok {
			return []types.PathMatch{
				{
					Path:          bob.GetCurrentPath(instance),
					Orchestration: orch,
					Depth:         0, // Leaf nodes have depth 0
					Type:          bob.orchestrationType,
				},
			}
		}
	}
	return []types.PathMatch{}
}

// GetOrchestrationTree returns a tree representation of the orchestration.
// For leaf orchestrations, this is a single-node tree.
// Container orchestrations should override this to include children.
func (bob *BaseOrchestrationBuilder) GetOrchestrationTree(instance any) *types.OrchestrationTree {
	if orch, ok := instance.(types.Orchestration); ok {
		return &types.OrchestrationTree{
			Name:          bob.GetName(),
			Path:          bob.GetCurrentPath(instance),
			Type:          bob.orchestrationType,
			Depth:         0,
			Orchestration: orch,
			Children:      []*types.OrchestrationTree{}, // Leaf nodes have no children
		}
	}
	return nil
}

// Query returns a PathQuery instance for advanced path-based queries.
// This method works for all orchestration types.
func (bob *BaseOrchestrationBuilder) Query(instance any) *types.PathQuery {
	tree := bob.GetOrchestrationTree(instance)
	return types.NewPathQuery(tree)
}
