// Package sequential provides sequential orchestration for the orchestrator library.
// It includes the SequentialBuilder for creating and executing sequential workflows
// with comprehensive error handling, atomic status management, and hierarchical configuration.
//
// # Sequential Execution
//
// Sequential orchestrations execute child orchestrations one after another in order.
// Each orchestration must complete before the next one begins. This provides:
//
//   - Predictable execution order
//   - Dependency management between operations
//   - Early termination on errors (with FailFast strategy)
//   - Complete error collection (with CollectAll strategy)
//
// # Error Handling Strategies
//
// Sequential orchestrations support two error handling strategies:
//
//   - FailFast: Stop execution on first error, cancel remaining operations
//   - CollectAll: Continue execution, collect all errors for comprehensive reporting
//
// # Configuration Inheritance
//
// Sequential orchestrations support hierarchical configuration:
//
//	parent := Config{Timeout: 30*time.Second, ErrorStrategy: FailFast}
//	sequential := Sequential(tasks...).With(Config{Timeout: 10*time.Second})
//	// Child inherits ErrorStrategy but overrides Timeout
//
// # Thread Safety
//
// All sequential operations are thread-safe using atomic operations for status management.
// Sequential orchestrations can be safely accessed from multiple goroutines concurrently.
//
// # Performance Characteristics
//
// The sequential execution engine is designed for efficiency:
//   - Zero-allocation status operations using atomic primitives
//   - Minimal memory overhead with object pooling support
//   - Efficient error collection and propagation
//   - Lock-free status management
//
// # Usage Examples
//
//	// Basic sequential execution
//	result, err := Sequential(
//	    Task(fetchUser),
//	    Task(validateUser),
//	    Task(processUser),
//	).Execute(ctx, config)
//
//	// Named sequential with configuration
//	result, err := Sequential(tasks...).
//	    Named("user-processing-pipeline").
//	    With(Config{Timeout: 60*time.Second}).
//	    ErrorBoundary(CollectAll).
//	    Execute(ctx, config)
//
//	// Nested sequential orchestrations
//	result, err := Sequential(
//	    Sequential(setupTasks...),
//	    Sequential(processingTasks...),
//	    Sequential(cleanupTasks...),
//	).Execute(ctx, config)
package sequential

import (
	"context"
	"fmt"
	"time"

	orchestration "github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/pkg/config"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/result"
	"github.com/maniartech/orchestrator/pkg/types"
)

// SequentialBuilder provides a fluent API for creating and configuring sequential orchestrations.
type SequentialBuilder struct {
	// Embed the container base to implement ContainerOrchestration and centralize child management
	*orchestration.BaseContainerOrchestration

	// TEMP: Keep the local slice for minimal-churn; kept in sync with base during construction.
	// TODO: Remove and replace all usages with BaseContainerOrchestration.GetChildren().
	orchestrations []types.Orchestration

	// Hierarchical naming
	namer         *types.HierarchicalNamer // Hierarchical naming system
	parentContext *types.NamingContext     // Parent naming context for nested orchestrations
}

// Sequential creates a new SequentialBuilder with the provided orchestrations.
// The orchestrations will be executed in the order they are provided.
// This is the primary constructor for creating sequential workflows.
//
// Parameters:
//   - orchestrations: Variable number of orchestrations to execute sequentially
//
// Returns:
//   - *SequentialBuilder: A new sequential builder instance
//
// Panics:
//   - If no orchestrations are provided
//   - If any orchestration is nil
//
// Example:
//
//	// Simple sequential execution
//	seq := Sequential(
//	    Task(func(ctx orchContext.Context) (string, error) { return "step1", nil }),
//	    Task(func(ctx orchContext.Context) (string, error) { return "step2", nil }),
//	    Task(func(ctx orchContext.Context) (string, error) { return "step3", nil }),
//	)
//
//	// Sequential with different orchestration types
//	seq := Sequential(
//	    Task(fetchUserData),
//	    Concurrent(validateData, enrichData),
//	    Task(saveUserData),
//	)
func Sequential(orchestrations ...types.Orchestration) *SequentialBuilder {
	if len(orchestrations) == 0 {
		panic("sequential: at least one orchestration is required")
	}
	// Validate all orchestrations are non-nil
	for i, orch := range orchestrations {
		if orch == nil {
			panic(fmt.Sprintf("sequential: orchestration at index %d is nil", i))
		}
	}

	// Initialize the container base with the children list
	sb := &SequentialBuilder{
		BaseContainerOrchestration: orchestration.NewBaseContainerOrchestration("sequential", orchestrations),
		orchestrations:             orchestrations, // TEMP: keep in sync for existing code paths
	}

	// Initialize hierarchical naming system
	sb.initializeNaming()

	// Wire path resolver callbacks on the base resolver
	sb.GetPathResolver().SetCallbacks(
		func() string {
			if sb.namer != nil {
				return sb.namer.GetContext().GetPath()
			}
			return ""
		},
		func() []types.Orchestration { return sb.GetChildren() },
		func(child types.Orchestration, index int) string { return sb.getChildName(child, index) },
	)

	return sb
}

// initializeNaming initializes the hierarchical naming system for this sequential builder.
// This method sets up the naming context and hierarchical namer.
func (sb *SequentialBuilder) initializeNaming() {
	// Initialize with default naming (will be updated if Named() is called)
	sb.namer = types.NewHierarchicalNamer(sb.parentContext, sb.GetName(), "sequential", 0)
}

// SetParentContext sets the parent naming context for nested orchestrations.
// This method is used internally when this sequential is used as a child orchestration.
func (sb *SequentialBuilder) SetParentContext(parentContext *types.NamingContext, index int) {
	sb.parentContext = parentContext
	sb.namer = types.NewHierarchicalNamer(parentContext, sb.GetName(), "sequential", index)
}

// Named sets a name for the sequential orchestration for observability and debugging.
// The name appears in logs and error messages to help identify which orchestration failed.
// Returns the same SequentialBuilder instance for method chaining.
//
// Example:
//
//	seq := Sequential(tasks...).Named("user-processing-pipeline")
func (sb *SequentialBuilder) Named(name string) types.Orchestration {
	sb.SetName(name)
	// Refresh the naming system with the new name
	sb.initializeNaming()
	return sb
}

// GetType returns the sequential orchestration type for debugging and observability.
func (sb *SequentialBuilder) GetType() string {
	return sb.BaseOrchestrationBuilder.GetType()
}

// With applies configuration to the sequential orchestration.
// Configuration is inherited hierarchically with local overrides.
// Child orchestrations inherit this configuration unless they specify their own.
// Returns the same SequentialBuilder instance for method chaining.
//
// Example:
//
//	seq := Sequential(tasks...).
//	    With(config.Config{
//	        Timeout: 60*time.Second,
//	        ErrorStrategy: errors.CollectAll,
//	    })
func (sb *SequentialBuilder) With(config config.Config) types.Orchestration {
	sb.SetConfig(config)
	return sb
}

// ErrorBoundary sets error handling strategy for this sequential orchestration.
// This controls how errors propagate within this orchestration scope.
// Returns the same SequentialBuilder instance for method chaining.
//
// Supported strategies:
//   - FailFast: Stop execution on first error, cancel remaining operations
//   - CollectAll: Continue execution, collect all errors
//
// Example:
//
//	seq := Sequential(tasks...).ErrorBoundary(errors.CollectAll)
func (sb *SequentialBuilder) ErrorBoundary(strategy errors.ErrorStrategy) types.Orchestration {
	sb.SetErrorBoundary(strategy)
	return sb
}

// Execute runs the sequential orchestration with the provided context and configuration.
// This method implements the core sequential execution logic with comprehensive error handling,
// configuration inheritance, and proper resource management.
//
// The execution process:
// 1. Applies configuration inheritance and error boundary settings
// 2. Executes orchestrations sequentially in order
// 3. Handles errors according to the configured strategy
// 4. Collects and aggregates results
// 5. Returns comprehensive results with error information
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - config: Base configuration to inherit from
//
// Returns:
//   - *result.Result: Aggregated results from all orchestrations
//   - error: First error encountered (FailFast) or aggregated errors (CollectAll)
//
// Example:
//
//	result, err := sequential.Execute(ctx, config)
//	if err != nil {
//	    log.Printf("Sequential execution failed: %v", err)
//	}
//
//	// Access individual results by orchestration name
//	userData := result.Get("fetch-user")
//	processedData := result.Get("process-user")
func (sb *SequentialBuilder) Execute(ctx context.Context, config config.Config) (*result.Result, error) {
	// Ensure sequential can only be executed once
	if !sb.CompareAndSwapStatus(types.NotStarted, types.Running) {
		return nil, fmt.Errorf("sequential orchestration already executed or in progress, current status: %v", sb.GetStatus())
	}

	startTime := time.Now()

	// Apply configuration inheritance
	finalConfig := config
	if sb.GetConfig() != nil {
		finalConfig = sb.GetConfig().Inherit(config)
	}

	// Apply error boundary if specified
	if sb.GetErrorBoundary() != nil {
		finalConfig.ErrorStrategy = *sb.GetErrorBoundary()
	}

	// Create shared orchestrator context for data sharing between tasks
	if finalConfig.OrchestrationContext == nil {
		orchCtx := orchContext.NewContext(finalConfig)
		finalConfig.OrchestrationContext = orchCtx
	}

	// Create result container
	result := result.NewResult()

	// Execute orchestrations sequentially based on error strategy
	var executionError error
	switch finalConfig.ErrorStrategy {
	case errors.FailFast:
		executionError = sb.executeFailFast(ctx, finalConfig, result)
	case errors.CollectAll:
		executionError = sb.executeCollectAll(ctx, finalConfig, result)
	default:
		executionError = sb.executeFailFast(ctx, finalConfig, result) // Default to FailFast
	}

	// Update status based on outcome
	if executionError != nil {
		if ctx.Err() != nil {
			sb.SetStatus(types.Cancelled)
		} else {
			sb.SetStatus(types.Completed)
		}
	} else {
		sb.SetStatus(types.Completed)
	}

	// Add execution metadata if there were errors
	if executionError != nil && len(result.Errors()) == 0 {
		// Add a sequential-level error if no child errors were collected
		result.AddError(errors.OperationError{
			Error:     executionError,
			Index:     -1, // Sequential-level error
			Duration:  time.Since(startTime),
			Timestamp: startTime,
			OpID:      sb.GetOperationID(),
		})
	}

	return result, executionError
}

// executeFailFast implements fail-fast execution strategy with enhanced error handling.
// Stops execution immediately on the first error and cancels remaining operations.
// This provides fast failure detection, minimal resource usage, and rich error context.
func (sb *SequentialBuilder) executeFailFast(ctx context.Context, config config.Config, result *result.Result) error {
	// Create error boundary handler for fail-fast strategy
	boundaryName := sb.GetOperationID()
	errorHandler := errors.NewErrorBoundaryHandler(errors.FailFast, boundaryName, nil)

	// Create rich error context
	errorContext := errors.CreateErrorContext(
		sb.GetName(),
		sb.GetOperationID(),
		"sequential",
		len(sb.orchestrations),
		config.ErrorStrategy,
		sb.GetErrorBoundary(),
	)

	// Set up panic recovery within error boundary
	defer func() {
		if panicErr := errorHandler.HandlePanic(); panicErr != nil {
			// Add panic error to result
			result.AddError(errors.OperationError{
				Error:     panicErr,
				Index:     -1,
				Duration:  0,
				Timestamp: time.Now(),
				OpID:      fmt.Sprintf("%s.panic", boundaryName),
				Stack:     errorHandler.CaptureStackTrace(),
			})
		}
	}()

	executionStart := time.Now()

	for i, orch := range sb.orchestrations {
		// Check for cancellation before each orchestration
		select {
		case <-ctx.Done():
			errors.UpdateErrorContext(errorContext, i, time.Since(executionStart), true)
			return ctx.Err()
		default:
		}

		stepStart := time.Now()
		orchResult, err := orch.Execute(ctx, config)
		stepDuration := time.Since(stepStart)
		stepName := sb.getChildName(orch, i)

		// Update error context with current progress
		errors.UpdateErrorContext(errorContext, i, time.Since(executionStart), false)

		// Handle error through error boundary
		if err != nil {
			errors.SetFailedStep(errorContext, i, stepName, errorHandler.CaptureStackTrace())

			shouldContinue := errorHandler.HandleError(err, i, stepName, stepDuration, errorContext)

			// Add error to result with rich context
			result.AddError(errors.OperationError{
				Error:     err,
				Index:     i,
				Duration:  stepDuration,
				Timestamp: stepStart,
				OpID:      sb.getChildOperationID(orch, i),
				Stack:     errorHandler.CaptureStackTrace(),
			})

			if !shouldContinue {
				// Generate enhanced error report for debugging
				reporter := errors.NewEnhancedErrorReporting(errorContext, errorHandler)
				enhancedError := fmt.Errorf("sequential orchestration failed at step %d (%s): %w\n\nDetailed Report:\n%s",
					i, stepName, err, reporter.GenerateErrorReport())

				return enhancedError
			}
		}

		// Store successful result using the child name
		if orchResult != nil {
			// First merge the result to preserve any nested results
			result.Merge(orchResult)

			// Then store the main result using the child name for direct access
			// Try to get the result using the child's own name first, then fallback to "task_result"
			var taskResult any
			if orch.GetName() != "" {
				taskResult = orchResult.Get(orch.GetName())
			}
			if taskResult == nil {
				taskResult = orchResult.Get("task_result")
			}

			if taskResult != nil {
				result.Set(stepName, taskResult)
			}
		}
	}

	return nil
}

// executeCollectAll implements collect-all execution strategy with enhanced error handling.
// Continues execution even when errors occur and collects all errors for comprehensive reporting.
// This provides complete error visibility, maximum operation completion, and rich error context.
func (sb *SequentialBuilder) executeCollectAll(ctx context.Context, config config.Config, result *result.Result) error {
	// Create error boundary handler for collect-all strategy
	boundaryName := sb.GetOperationID()
	errorHandler := errors.NewErrorBoundaryHandler(errors.CollectAll, boundaryName, nil)

	// Create rich error context
	errorContext := errors.CreateErrorContext(
		sb.GetName(),
		sb.GetOperationID(),
		"sequential",
		len(sb.orchestrations),
		config.ErrorStrategy,
		sb.GetErrorBoundary(),
	)

	// Set up panic recovery within error boundary
	defer func() {
		if panicErr := errorHandler.HandlePanic(); panicErr != nil {
			// Add panic error to result
			result.AddError(errors.OperationError{
				Error:     panicErr,
				Index:     -1,
				Duration:  0,
				Timestamp: time.Now(),
				OpID:      fmt.Sprintf("%s.panic", boundaryName),
				Stack:     errorHandler.CaptureStackTrace(),
			})
		}
	}()

	executionStart := time.Now()

	for i, orch := range sb.orchestrations {
		// Check for cancellation before each orchestration
		select {
		case <-ctx.Done():
			errors.UpdateErrorContext(errorContext, i, time.Since(executionStart), true)

			// Add cancellation error for remaining operations with rich context
			for j := i; j < len(sb.orchestrations); j++ {
				stepName := sb.getChildName(sb.orchestrations[j], j)
				cancellationErr := ctx.Err()

				errorHandler.HandleError(cancellationErr, j, stepName, 0, errorContext)

				result.AddError(errors.OperationError{
					Error:     cancellationErr,
					Index:     j,
					Duration:  0,
					Timestamp: time.Now(),
					OpID:      sb.getChildOperationID(sb.orchestrations[j], j),
					Stack:     errorHandler.CaptureStackTrace(),
				})
			}
			return ctx.Err()
		default:
		}

		stepStart := time.Now()
		orchResult, err := orch.Execute(ctx, config)
		stepDuration := time.Since(stepStart)
		stepName := sb.getChildName(orch, i)

		// Update error context with current progress
		errors.UpdateErrorContext(errorContext, i+1, time.Since(executionStart), false)

		// Handle error through error boundary (CollectAll continues execution)
		if err != nil {
			errors.SetFailedStep(errorContext, i, stepName, errorHandler.CaptureStackTrace())

			errorHandler.HandleError(err, i, stepName, stepDuration, errorContext)

			// Add error to result with rich context
			result.AddError(errors.OperationError{
				Error:     err,
				Index:     i,
				Duration:  stepDuration,
				Timestamp: stepStart,
				OpID:      sb.getChildOperationID(orch, i),
				Stack:     errorHandler.CaptureStackTrace(),
			})
		} else {
			// Store successful result using the child name
			if orchResult != nil {
				// First merge the result to preserve any nested results
				result.Merge(orchResult)

				// Then store the main result using the child name for direct access
				// Try to get the result using the child's own name first, then fallback to "task_result"
				var taskResult any
				if orch.GetName() != "" {
					taskResult = orchResult.Get(orch.GetName())
				}
				if taskResult == nil {
					taskResult = orchResult.Get("task_result")
				}

				if taskResult != nil {
					result.Set(stepName, taskResult)
				}
			}
		}
	}

	// Return enhanced error if any errors occurred
	if errorHandler.HasErrors() {
		// Generate comprehensive error report
		reporter := errors.NewEnhancedErrorReporting(errorContext, errorHandler)
		enhancedError := fmt.Errorf("sequential orchestration completed with %d errors: %w\n\nDetailed Report:\n%s",
			errorHandler.GetErrorCount(), errorHandler.GetFinalError(), reporter.GenerateErrorReport())

		return enhancedError
	}

	return nil
}

// getChildName returns a descriptive name for a child orchestration.
// Uses the orchestration's name if available, otherwise generates a consistent name.
func (sb *SequentialBuilder) getChildName(orch types.Orchestration, index int) string {
	if name := orch.GetName(); name != "" {
		return name
	}

	// Generate a consistent name using the index directly
	// This ensures step numbers match the position in the sequence
	return fmt.Sprintf("step-%d", index)
}

// getChildOperationID generates a unique operation ID for a child orchestration.
// Combines sequential name with child information for traceability.
func (sb *SequentialBuilder) getChildOperationID(orch types.Orchestration, index int) string {
	sequentialID := sb.GetOperationID()
	childName := sb.getChildName(orch, index)
	return fmt.Sprintf("%s.%s", sequentialID, childName)
}

// GetOperationID returns a unique operation ID for this sequential orchestration.
// This provides a standardized way to access the operation ID without needing to pass the instance.
func (sb *SequentialBuilder) GetOperationID() string {
	// BaseContainerOrchestration embeds BaseOrchestrationBuilder,
	// so we can still delegate to it for the concrete instance op-id.
	return sb.BaseOrchestrationBuilder.GetOperationID(sb)
}

// ContainerOrchestration delegations (optional now; can remove duplicates later)
// These help migrate away from sb.orchestrations usages.
func (sb *SequentialBuilder) GetChildAt(index int) (types.Orchestration, error) {
	return sb.BaseContainerOrchestration.GetChildAt(index)
}
func (sb *SequentialBuilder) GetChildCount() int {
	return sb.BaseContainerOrchestration.GetChildCount()
}
func (sb *SequentialBuilder) GetChildren() []types.Orchestration {
	return sb.BaseContainerOrchestration.GetChildren()
}
func (sb *SequentialBuilder) GetChildByName(name string) (types.Orchestration, error) {
	return sb.BaseContainerOrchestration.GetChildByName(name)
}
func (sb *SequentialBuilder) GetFirstChild() (types.Orchestration, error) {
	return sb.BaseContainerOrchestration.GetFirstChild()
}
func (sb *SequentialBuilder) GetLastChild() (types.Orchestration, error) {
	return sb.BaseContainerOrchestration.GetLastChild()
}
func (sb *SequentialBuilder) FindChildByName(name string) (int, types.Orchestration) {
	// Override to support generated names like "step-<index>" in addition to explicit names
	children := sb.GetChildren()
	for i, child := range children {
		if sb.getChildName(child, i) == name {
			return i, child
		}
	}
	return -1, nil
}
func (sb *SequentialBuilder) ContainsChild(child types.Orchestration) bool {
	return sb.BaseContainerOrchestration.ContainsChild(child)
}
func (sb *SequentialBuilder) ContainsChildWithName(name string) bool {
	return sb.BaseContainerOrchestration.ContainsChildWithName(name)
}
func (sb *SequentialBuilder) GetChildByOperationID(opID string) (types.Orchestration, error) {
	return sb.BaseContainerOrchestration.GetChildByOperationID(opID)
}

// GetChildNames returns the names of all child orchestrations, using explicit names
// if set or generated names (step-<index>) otherwise. This keeps naming stable
// before and after execution and is used by unit tests.
func (sb *SequentialBuilder) GetChildNames() []string {
	children := sb.GetChildren()
	names := make([]string, len(children))
	for i, child := range children {
		names[i] = sb.getChildName(child, i)
	}
	return names
}

// =============================================================================
// PathResolver Interface Implementation
// =============================================================================

// GetByPath finds an orchestration by its hierarchical path using dynamic resolution.
// This method allows finding any orchestration in the tree without maintaining a centralized map.
//
// Parameters:
//   - path: Hierarchical path (e.g., "main-pipeline.auth-flow.validate-user")
//
// Returns:
//   - orchestration.Orchestration: Found orchestration (nil if not found)
//   - error: Error if path is invalid or orchestration not found
//
// Example:
//
//	// Find a deeply nested task
//	task, err := sequential.GetByPath("main-pipeline.auth-flow.validate-user")
//	if err != nil {
//	    log.Printf("Task not found: %v", err)
//	} else {
//	    log.Printf("Found task: %s", task.GetName())
//	}
func (sb *SequentialBuilder) GetByPath(path string) (types.Orchestration, error) {
	return sb.BaseContainerOrchestration.GetByPath(path)
}

// GetCurrentPath returns the current orchestration's full hierarchical path.
// Each orchestration knows its own path without requiring a centralized registry.
func (sb *SequentialBuilder) GetCurrentPath() string {
	return sb.BaseContainerOrchestration.GetCurrentPath()
}

// ListAllPaths returns all available paths in the orchestration subtree.
// This method performs a depth-first traversal to collect all paths dynamically.
//
// Returns:
//   - []string: All paths in the subtree
//
// Example:
//
//	paths := sequential.ListAllPaths()
//	for _, path := range paths {
//	    log.Printf("Available path: %s", path)
//	}
func (sb *SequentialBuilder) ListAllPaths() []string {
	return sb.BaseContainerOrchestration.ListAllPaths()
}

// FindByName searches for orchestrations by name across the entire subtree.
// This method can return multiple matches if the same name appears at different levels.
//
// Parameters:
//   - name: Name to search for
//
// Returns:
//   - []orchestration.PathMatch: All matching orchestrations with their path information
//
// Example:
//
//	matches := sequential.FindByName("validate-user")
//	for _, match := range matches {
//	    log.Printf("Found '%s' at path: %s (depth: %d)", name, match.Path, match.Depth)
//	}
func (sb *SequentialBuilder) FindByName(name string) []types.PathMatch {
	return sb.BaseContainerOrchestration.FindByName(name)
}

// =============================================================================
// Advanced Path Query Methods
// =============================================================================

// Query returns a PathQuery instance for advanced path-based queries.
// This provides pattern matching, type-based searches, and other advanced features.
//
// Returns:
//   - *orchestration.PathQuery: Query instance for advanced operations
//
// Example:
//
//	query := sequential.Query()
//	authTasks := query.FindByPattern("*.auth.*")
//	sequentialOrchestrations := query.FindByType("sequential")
func (sb *SequentialBuilder) Query() *types.PathQuery {
	return sb.BaseContainerOrchestration.Query()
}

// GetOrchestrationTree returns a tree representation of the orchestration hierarchy.
// This method provides a structured view of the entire orchestration tree.
//
// Returns:
//   - *orchestration.OrchestrationTree: Tree representation
//
// Example:
//
//	tree := sequential.GetOrchestrationTree()
//	tree.Print() // Prints the tree structure
func (sb *SequentialBuilder) GetOrchestrationTree() *types.OrchestrationTree {
	return sb.BaseContainerOrchestration.GetOrchestrationTree()
}
