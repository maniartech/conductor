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
	"sync/atomic"
	"time"

	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/result"
	"github.com/maniartech/orchestrator/types"
)

// SequentialBuilder provides a fluent API for creating and configuring sequential orchestrations.
// It executes child orchestrations one after another in order, with comprehensive error handling
// and configuration inheritance following Go best practices.
//
// Example:
//
//	sequential := Sequential(
//	    Task(fetchData),
//	    Task(processData),
//	    Task(saveData),
//	).Named("data-pipeline").
//	With(config.Config{Timeout: 30*time.Second})
type SequentialBuilder struct {
	orchestrations []types.Orchestration
	name           string
	config         *config.Config
	errorBoundary  *errors.ErrorStrategy
	status         atomic.Uint32                   // Atomic status management
	namer          *types.HierarchicalNamer        // Hierarchical naming system
	parentContext  *types.NamingContext            // Parent naming context for nested orchestrations
	pathResolver   *orchestration.PathResolverBase // Path-based orchestration resolution
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
//	    Task(func() (string, error) { return "step1", nil }),
//	    Task(func() (string, error) { return "step2", nil }),
//	    Task(func() (string, error) { return "step3", nil }),
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
		panic("sequential orchestration requires at least one orchestration")
	}

	// Validate all orchestrations are non-nil
	for i, orch := range orchestrations {
		if orch == nil {
			panic(fmt.Sprintf("orchestration at index %d cannot be nil", i))
		}
	}

	sb := &SequentialBuilder{
		orchestrations: orchestrations,
	}

	// Initialize hierarchical naming system
	// This will be updated when Named() is called or when parent context is set
	sb.initializeNaming()

	// Initialize path resolver
	sb.pathResolver = orchestration.NewPathResolverBase()
	sb.pathResolver.SetCallbacks(
		func() string { // avoid recursive call through pathResolver
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
	sb.namer = types.NewHierarchicalNamer(sb.parentContext, sb.name, "sequential", 0)
}

// SetParentContext sets the parent naming context for nested orchestrations.
// This method is used internally when this sequential is used as a child orchestration.
func (sb *SequentialBuilder) SetParentContext(parentContext *types.NamingContext, index int) {
	sb.parentContext = parentContext
	sb.namer = types.NewHierarchicalNamer(parentContext, sb.name, "sequential", index)
}

// Named sets a name for the sequential orchestration for observability and debugging.
// The name appears in logs and error messages to help identify which orchestration failed.
// Returns the same SequentialBuilder instance for method chaining.
//
// Example:
//
//	seq := Sequential(tasks...).Named("user-processing-pipeline")
func (sb *SequentialBuilder) Named(name string) types.Orchestration {
	sb.name = name
	// Refresh the naming system with the new name
	sb.initializeNaming()
	return sb
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
	sb.config = &config
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
	sb.errorBoundary = &strategy
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
	if !sb.compareAndSwapStatus(orchestration.NotStarted, orchestration.Running) {
		return nil, fmt.Errorf("sequential orchestration already executed or in progress, current status: %v", sb.GetStatus())
	}

	startTime := time.Now()

	// Apply configuration inheritance
	finalConfig := config
	if sb.config != nil {
		finalConfig = sb.config.Inherit(config)
	}

	// Apply error boundary if specified
	if sb.errorBoundary != nil {
		finalConfig.ErrorStrategy = *sb.errorBoundary
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
			sb.setStatus(types.Cancelled)
		} else {
			sb.setStatus(types.Completed)
		}
	} else {
		sb.setStatus(types.Completed)
	}

	// Add execution metadata if there were errors
	if executionError != nil && len(result.Errors()) == 0 {
		// Add a sequential-level error if no child errors were collected
		result.AddError(errors.OperationError{
			Error:     executionError,
			Index:     -1, // Sequential-level error
			Duration:  time.Since(startTime),
			Timestamp: startTime,
			OpID:      sb.getOperationID(),
		})
	}

	return result, executionError
}

// executeFailFast implements fail-fast execution strategy with enhanced error handling.
// Stops execution immediately on the first error and cancels remaining operations.
// This provides fast failure detection, minimal resource usage, and rich error context.
func (sb *SequentialBuilder) executeFailFast(ctx context.Context, config config.Config, result *result.Result) error {
	// Create error boundary handler for fail-fast strategy
	boundaryName := sb.getOperationID()
	errorHandler := errors.NewErrorBoundaryHandler(errors.FailFast, boundaryName, nil)

	// Create rich error context
	errorContext := errors.CreateErrorContext(
		sb.name,
		sb.getOperationID(),
		"sequential",
		len(sb.orchestrations),
		config.ErrorStrategy,
		sb.errorBoundary,
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
	boundaryName := sb.getOperationID()
	errorHandler := errors.NewErrorBoundaryHandler(errors.CollectAll, boundaryName, nil)

	// Create rich error context
	errorContext := errors.CreateErrorContext(
		sb.name,
		sb.getOperationID(),
		"sequential",
		len(sb.orchestrations),
		config.ErrorStrategy,
		sb.errorBoundary,
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
func (sb *SequentialBuilder) getChildOperationID(orch orchestration.Orchestration, index int) string {
	sequentialID := sb.getOperationID()
	childName := sb.getChildName(orch, index)
	return fmt.Sprintf("%s.%s", sequentialID, childName)
}

// getOperationID generates a unique operation ID for traceability.
// Uses the sequential name if available, otherwise generates a default ID.
func (sb *SequentialBuilder) getOperationID() string {
	if sb.name != "" {
		return fmt.Sprintf("sequential-%s", sb.name)
	}
	return fmt.Sprintf("sequential-%p", sb)
}

// GetName returns the sequential orchestration name for debugging and observability.
// Returns empty string if no name was set.
func (sb *SequentialBuilder) GetName() string {
	return sb.name
}

// GetConfig returns the sequential orchestration's configuration.
// Returns nil if no configuration was set.
func (sb *SequentialBuilder) GetConfig() *config.Config {
	return sb.config
}

// GetStatus returns the current sequential orchestration status using atomic operations.
// This method is thread-safe and can be called concurrently.
func (sb *SequentialBuilder) GetStatus() types.Status {
	return types.Status(sb.status.Load())
}

// GetChildAt returns the child orchestration at the specified index.
// This method provides access to individual child orchestrations for introspection,
// debugging, and dynamic orchestration management.
//
// Parameters:
//   - index: Zero-based index of the child orchestration to retrieve
//
// Returns:
//   - orchestration.Orchestration: The child orchestration at the specified index
//   - error: Error if index is out of bounds
//
// Example:
//
//	seq := Sequential(
//	    Task(func() (string, error) { return "task1", nil }).Named("first-task"),
//	    Task(func() (int, error) { return 42, nil }).Named("second-task"),
//	    Task(func() (bool, error) { return true, nil }).Named("third-task"),
//	)
//
//	// Access the second child (index 1)
//	child, err := seq.GetChildAt(1)
//	if err != nil {
//	    log.Printf("Error accessing child: %v", err)
//	} else {
//	    log.Printf("Child name: %s", child.GetName()) // "second-task"
//	}
func (sb *SequentialBuilder) GetChildAt(index int) (types.Orchestration, error) {
	if index < 0 || index >= len(sb.orchestrations) {
		return nil, fmt.Errorf("index %d out of bounds: sequential has %d orchestrations (valid range: 0-%d)",
			index, len(sb.orchestrations), len(sb.orchestrations)-1)
	}
	return sb.orchestrations[index], nil
}

// GetChildCount returns the total number of child orchestrations in this sequential.
// This method provides the count of orchestrations that will be executed sequentially.
//
// Returns:
//   - int: The number of child orchestrations
//
// Example:
//
//	seq := Sequential(task1, task2, task3)
//	count := seq.GetChildCount() // Returns 3
func (sb *SequentialBuilder) GetChildCount() int {
	return len(sb.orchestrations)
}

// GetChildren returns a copy of all child orchestrations.
// This method provides access to all child orchestrations for introspection and analysis.
// Returns a copy to prevent external modification of the internal orchestration slice.
//
// Returns:
//   - []orchestration.Orchestration: Copy of all child orchestrations
//
// Example:
//
//	seq := Sequential(task1, task2, task3)
//	children := seq.GetChildren()
//	for i, child := range children {
//	    log.Printf("Child %d: %s (status: %v)", i, child.GetName(), child.GetStatus())
//	}
func (sb *SequentialBuilder) GetChildren() []types.Orchestration {
	// Return a copy to prevent external modification
	children := make([]types.Orchestration, len(sb.orchestrations))
	copy(children, sb.orchestrations)
	return children
}

// FindChildByName searches for a child orchestration by name and returns its index and the orchestration.
// This method provides a convenient way to locate specific child orchestrations by their names.
//
// Parameters:
//   - name: The name of the child orchestration to find
//
// Returns:
//   - int: The index of the found orchestration (-1 if not found)
//   - orchestration.Orchestration: The found orchestration (nil if not found)
//
// Example:
//
//	seq := Sequential(
//	    Task(func() (string, error) { return "data", nil }).Named("fetch-data"),
//	    Task(func() (string, error) { return "processed", nil }).Named("process-data"),
//	)
//
//	index, child := seq.FindChildByName("process-data")
//	if child != nil {
//	    log.Printf("Found 'process-data' at index %d", index) // index = 1
//	} else {
//	    log.Println("Child not found")
//	}
func (sb *SequentialBuilder) FindChildByName(name string) (int, types.Orchestration) {
	for i, orch := range sb.orchestrations {
		if sb.getChildName(orch, i) == name {
			return i, orch
		}
	}
	return -1, nil
}

// GetChildNames returns the names of all child orchestrations.
// This method provides a convenient way to get an overview of all child orchestration names.
// Unnamed orchestrations are represented with generated names (e.g., "step-0", "step-1").
//
// Returns:
//   - []string: Slice of child orchestration names
//
// Example:
//
//	seq := Sequential(
//	    Task(func() (string, error) { return "data", nil }).Named("fetch-data"),
//	    Task(func() (string, error) { return "processed", nil }), // unnamed
//	    Task(func() (string, error) { return "saved", nil }).Named("save-data"),
//	)
//
//	names := seq.GetChildNames()
//	// Returns: ["fetch-data", "step-1", "save-data"]
func (sb *SequentialBuilder) GetChildNames() []string {
	names := make([]string, len(sb.orchestrations))
	for i, orch := range sb.orchestrations {
		names[i] = sb.getChildName(orch, i)
	}
	return names
}

// setStatus atomically sets the sequential orchestration status.
// This is an internal method used during sequential execution.
func (sb *SequentialBuilder) setStatus(status types.Status) {
	sb.status.Store(uint32(status))
}

// compareAndSwapStatus atomically compares and swaps the sequential orchestration status.
// Returns true if the swap was successful, false otherwise.
// This ensures thread-safe status transitions.
func (sb *SequentialBuilder) compareAndSwapStatus(old, new orchestration.Status) bool {
	return sb.status.CompareAndSwap(uint32(old), uint32(new))
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
	// Handle self-reference
	if path == sb.GetCurrentPath() {
		return sb, nil
	}
	return sb.pathResolver.GetByPath(path)
}

// GetCurrentPath returns the current orchestration's full hierarchical path.
// Each orchestration knows its own path without requiring a centralized registry.
func (sb *SequentialBuilder) GetCurrentPath() string {
	if sb.namer != nil {
		return sb.namer.GetContext().GetPath()
	}
	return ""
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
	return sb.pathResolver.ListAllPaths()
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
	matches := sb.pathResolver.FindByName(name)

	// Check if current orchestration matches
	if sb.GetName() == name {
		currentMatch := orchestration.PathMatch{
			Path:          sb.GetCurrentPath(),
			Orchestration: sb,
			Depth:         sb.namer.GetDepth(),
			Parent:        sb.namer.GetContext().GetParentPath(),
			Type:          "sequential",
		}
		matches = append([]orchestration.PathMatch{currentMatch}, matches...)
	}

	return matches
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
	tree := sb.GetOrchestrationTree()
	return types.NewPathQuery(tree)
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
	tree := &types.OrchestrationTree{
		Name:          sb.GetName(),
		Path:          sb.GetCurrentPath(),
		Type:          "sequential",
		Depth:         sb.namer.GetDepth(),
		Orchestration: sb,
		Children:      make([]*types.OrchestrationTree, 0, len(sb.orchestrations)),
	}

	// Add children to tree
	for i, child := range sb.orchestrations {
		childName := sb.getChildName(child, i)
		childPath := sb.GetCurrentPath() + "." + childName

		childTree := &types.OrchestrationTree{
			Name:          childName,
			Path:          childPath,
			Type:          "unknown", // Will be determined by child type
			Depth:         sb.namer.GetDepth() + 1,
			Parent:        tree,
			Orchestration: child,
			Children:      []*orchestration.OrchestrationTree{},
		}

		// If child implements PathResolver, get its tree recursively
		if pathResolver, ok := child.(types.PathResolver); ok {
			childTree = pathResolver.GetOrchestrationTree()
			childTree.Parent = tree
		}

		tree.Children = append(tree.Children, childTree)
	}

	return tree
}
