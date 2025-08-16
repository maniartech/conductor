// Package concurrent provides concurrent orchestration for the orchestrator library.
// It includes the ConcurrentBuilder for creating and executing concurrent workflows
// with comprehensive error handling, atomic status management, and hierarchical configuration.
//
// # Concurrent Execution
//
// Concurrent orchestrations execute child orchestrations simultaneously using goroutines.
// All orchestrations run in parallel with proper synchronization and resource management:
//
//   - True parallelism with goroutine-based execution
//   - Bounded concurrency to prevent resource exhaustion
//   - WaitGroup-based synchronization for proper completion
//   - Context-based cancellation for graceful shutdown
//   - Semaphore-based resource control
//
// # Error Handling Strategies
//
// Concurrent orchestrations support two error handling strategies:
//
//   - FailFast: Cancel all operations on first error, return immediately
//   - CollectAll: Continue all operations, collect all errors for comprehensive reporting
//
// # Configuration Inheritance
//
// Concurrent orchestrations support hierarchical configuration:
//
//	parent := Config{Timeout: 30*time.Second, ErrorStrategy: FailFast}
//	concurrent := Concurrent(tasks...).With(Config{MaxConcurrency: 10})
//	// Child inherits ErrorStrategy and Timeout but overrides MaxConcurrency
//
// # Thread Safety
//
// All concurrent operations are thread-safe using atomic operations for status management.
// Concurrent orchestrations can be safely accessed from multiple goroutines concurrently.
//
// # Performance Characteristics
//
// The concurrent execution engine is designed for high performance:
//   - Zero-allocation status operations using atomic primitives
//   - Efficient goroutine lifecycle management with proper cleanup
//   - Bounded resource usage with semaphore-based concurrency control
//   - Lock-free error collection using atomic operations
//   - Minimal memory overhead with object pooling support
//
// # Usage Examples
//
//	// Basic concurrent execution
//	result, err := Concurrent(
//	    Task(fetchUserData),
//	    Task(fetchOrderData),
//	    Task(fetchInventoryData),
//	).Execute(ctx, config)
//
//	// Named concurrent with configuration
//	result, err := Concurrent(tasks...).
//	    Named("data-fetching-pipeline").
//	    With(Config{MaxConcurrency: 5, Timeout: 30*time.Second}).
//	    ErrorBoundary(CollectAll).
//	    Execute(ctx, config)
//
//	// Nested concurrent orchestrations
//	result, err := Concurrent(
//	    Concurrent(authTasks...),
//	    Concurrent(dataTasks...),
//	    Concurrent(validationTasks...),
//	).Execute(ctx, config)
package concurrent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/internal/result"
	"github.com/maniartech/orchestrator/types"
)

// ConcurrentBuilder provides a fluent API for creating and configuring concurrent orchestrations.
// It executes child orchestrations simultaneously using goroutines with comprehensive error handling,
// bounded concurrency control, and configuration inheritance following Go best practices.
//
// Example:
//
//	concurrent := Concurrent(
//	    Task(fetchUserData),
//	    Task(fetchOrderData),
//	    Task(fetchInventoryData),
//	).Named("data-pipeline").
//	With(config.Config{MaxConcurrency: 10, Timeout: 30*time.Second})
type ConcurrentBuilder struct {
	*orchestration.BaseOrchestrationBuilder
	orchestrations []types.Orchestration
	namer          *types.HierarchicalNamer        // Hierarchical naming system
	parentContext  *types.NamingContext            // Parent naming context for nested orchestrations
	pathResolver   *orchestration.PathResolverBase // Path-based orchestration resolution
}

// Concurrent creates a new ConcurrentBuilder with the provided orchestrations.
// The orchestrations will be executed simultaneously using goroutines.
// This is the primary constructor for creating concurrent workflows.
//
// Parameters:
//   - orchestrations: Variable number of orchestrations to execute concurrently
//
// Returns:
//   - *ConcurrentBuilder: A new concurrent builder instance
//
// Panics:
//   - If no orchestrations are provided
//   - If any orchestration is nil
//
// Example:
//
//	// Simple concurrent execution
//	conc := Concurrent(
//	    Task(func() (string, error) { return "task1", nil }),
//	    Task(func() (string, error) { return "task2", nil }),
//	    Task(func() (string, error) { return "task3", nil }),
//	)
//
//	// Concurrent with different orchestration types
//	conc := Concurrent(
//	    Task(fetchUserData),
//	    Sequential(validateData, enrichData),
//	    Task(saveUserData),
//	)
func Concurrent(orchestrations ...types.Orchestration) *ConcurrentBuilder {
	if len(orchestrations) == 0 {
		panic("concurrent orchestration requires at least one orchestration")
	}

	// Validate all orchestrations are non-nil
	for i, orch := range orchestrations {
		if orch == nil {
			panic(fmt.Sprintf("orchestration at index %d cannot be nil", i))
		}
	}

	cb := &ConcurrentBuilder{
		BaseOrchestrationBuilder: orchestration.NewBaseOrchestrationBuilder("concurrent"),
		orchestrations:           orchestrations,
	}

	// Initialize hierarchical naming system
	// This will be updated when Named() is called or when parent context is set
	cb.initializeNaming()

	// Initialize path resolver
	cb.pathResolver = orchestration.NewPathResolverBase()
	cb.pathResolver.SetCallbacks(
		func() string { return cb.GetCurrentPath() },
		func() []types.Orchestration { return cb.GetChildren() },
		func(child types.Orchestration, index int) string { return cb.getChildName(child, index) },
	)

	return cb
}

// initializeNaming initializes the hierarchical naming system for this concurrent builder.
// This method sets up the naming context and hierarchical namer.
func (cb *ConcurrentBuilder) initializeNaming() {
	// Initialize with default naming (will be updated if Named() is called)
	cb.namer = types.NewHierarchicalNamer(cb.parentContext, cb.GetName(), "concurrent", 0)
}

// SetParentContext sets the parent naming context for nested orchestrations.
// This method is used internally when this concurrent is used as a child orchestration.
func (cb *ConcurrentBuilder) SetParentContext(parentContext *types.NamingContext, index int) {
	cb.parentContext = parentContext
	cb.namer = types.NewHierarchicalNamer(parentContext, cb.GetName(), "concurrent", index)
}

// Named sets a name for the concurrent orchestration for observability and debugging.
// The name appears in logs and error messages to help identify which orchestration failed.
// Returns the same ConcurrentBuilder instance for method chaining.
//
// Example:
//
//	conc := Concurrent(tasks...).Named("data-processing-pipeline")
func (cb *ConcurrentBuilder) Named(name string) types.Orchestration {
	cb.SetName(name)
	// Refresh the naming system with the new name
	cb.initializeNaming()
	return cb
}

// With applies configuration to the concurrent orchestration.
// Configuration is inherited hierarchically with local overrides.
// Child orchestrations inherit this configuration unless they specify their own.
// Returns the same ConcurrentBuilder instance for method chaining.
//
// Example:
//
//	conc := Concurrent(tasks...).
//	    With(config.Config{
//	        MaxConcurrency: 10,
//	        Timeout: 60*time.Second,
//	        ErrorStrategy: errors.CollectAll,
//	    })
func (cb *ConcurrentBuilder) With(config config.Config) types.Orchestration {
	cb.SetConfig(config)
	return cb
}

// ErrorBoundary sets error handling strategy for this concurrent orchestration.
// This controls how errors propagate within this orchestration scope.
// Returns the same ConcurrentBuilder instance for method chaining.
//
// Supported strategies:
//   - FailFast: Cancel all operations on first error, return immediately
//   - CollectAll: Continue all operations, collect all errors
//
// Example:
//
//	conc := Concurrent(tasks...).ErrorBoundary(errors.CollectAll)
func (cb *ConcurrentBuilder) ErrorBoundary(strategy errors.ErrorStrategy) types.Orchestration {
	cb.SetErrorBoundary(strategy)
	return cb
}

// Execute runs the concurrent orchestration with the provided context and configuration.
// This method implements the core concurrent execution logic with comprehensive error handling,
// configuration inheritance, and proper resource management.
//
// The execution process:
// 1. Applies configuration inheritance and error boundary settings
// 2. Creates bounded concurrency control using semaphores
// 3. Executes orchestrations concurrently using goroutines
// 4. Handles errors according to the configured strategy
// 5. Collects and aggregates results with proper synchronization
// 6. Returns comprehensive results with error information
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
//	result, err := concurrent.Execute(ctx, config)
//	if err != nil {
//	    log.Printf("Concurrent execution failed: %v", err)
//	}
//
//	// Access individual results by orchestration name
//	userData := result.Get("fetch-user")
//	orderData := result.Get("fetch-orders")
func (cb *ConcurrentBuilder) Execute(ctx context.Context, config config.Config) (*result.Result, error) {
	// Validate execution preconditions (handled by base)
	if err := cb.ValidateExecutionPreconditions(); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Apply configuration inheritance (handled by base)
	finalConfig := cb.ApplyConfigurationInheritance(config)

	// Create result container
	result := result.NewResult()

	// Execute orchestrations concurrently based on error strategy
	var executionError error
	switch finalConfig.ErrorStrategy {
	case errors.FailFast:
		executionError = cb.executeFailFast(ctx, finalConfig, result)
	case errors.CollectAll:
		executionError = cb.executeCollectAll(ctx, finalConfig, result)
	default:
		executionError = cb.executeFailFast(ctx, finalConfig, result) // Default to FailFast
	}

	// Complete execution (handled by base)
	cb.CompleteExecution(ctx, executionError)

	// Add execution metadata if there were errors
	if executionError != nil && len(result.Errors()) == 0 {
		// Add a concurrent-level error if no child errors were collected
		result.AddError(errors.OperationError{
			Error:     executionError,
			Index:     -1, // Concurrent-level error
			Duration:  time.Since(startTime),
			Timestamp: startTime,
			OpID:      cb.GetOperationID(cb),
		})
	}

	return result, executionError
}

// executeFailFast implements fail-fast execution strategy with enhanced error handling.
// Cancels all operations immediately on the first error and returns with minimal resource usage.
// This provides fast failure detection and rich error context.
func (cb *ConcurrentBuilder) executeFailFast(ctx context.Context, config config.Config, result *result.Result) error {
	// Create cancellable context for fail-fast behavior
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create error collector for fail-fast strategy
	errorCollector := errors.NewErrorCollector(len(cb.orchestrations), errors.FailFast)

	// Create semaphore for concurrency control
	maxConcurrency := cb.getMaxConcurrency(config)
	semaphore := make(chan struct{}, maxConcurrency)

	// Create WaitGroup for synchronization
	var wg sync.WaitGroup

	// Execute all orchestrations concurrently
	for i, orch := range cb.orchestrations {
		wg.Add(1)
		go func(index int, orchestration types.Orchestration) {
			defer wg.Done()

			// Acquire semaphore for concurrency control
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				errorCollector.AddErrorWithOpID(index, ctx.Err(), 0, cb.getChildOperationID(orchestration, index))
				return
			}

			// Check if we should continue (fail-fast check)
			if errorCollector.ShouldStopExecution() {
				return // Early termination
			}

			// Execute orchestration
			stepStart := time.Now()
			orchResult, err := orchestration.Execute(ctx, config)
			stepDuration := time.Since(stepStart)
			stepName := cb.getChildName(orchestration, index)

			if err != nil {
				errorCollector.AddErrorWithOpID(index, err, stepDuration, cb.getChildOperationID(orchestration, index))
				cancel() // Cancel all other operations
			} else {
				// Store successful result
				cb.storeOrchestrationResult(result, orchResult, orchestration, stepName)
			}
		}(i, orch)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Return error if any occurred
	if errorCollector.HasErrors() {
		// Add all errors to result
		for _, opErr := range errorCollector.GetAllErrors() {
			result.AddError(opErr)
		}
		return errorCollector.GetFinalError()
	}

	return nil
}

// executeCollectAll implements collect-all execution strategy with enhanced error handling.
// Continues execution even when errors occur and collects all errors for comprehensive reporting.
// This provides complete error visibility and maximum operation completion.
func (cb *ConcurrentBuilder) executeCollectAll(ctx context.Context, config config.Config, result *result.Result) error {
	// Create error collector for collect-all strategy
	errorCollector := errors.NewErrorCollector(len(cb.orchestrations), errors.CollectAll)

	// Create semaphore for concurrency control
	maxConcurrency := cb.getMaxConcurrency(config)
	semaphore := make(chan struct{}, maxConcurrency)

	// Create WaitGroup for synchronization
	var wg sync.WaitGroup

	// Execute all orchestrations concurrently
	for i, orch := range cb.orchestrations {
		wg.Add(1)
		go func(index int, orchestration types.Orchestration) {
			defer wg.Done()

			// Acquire semaphore for concurrency control
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				errorCollector.AddErrorWithOpID(index, ctx.Err(), 0, cb.getChildOperationID(orchestration, index))
				return
			}

			// Execute orchestration (continue on error)
			stepStart := time.Now()
			orchResult, err := orchestration.Execute(ctx, config)
			stepDuration := time.Since(stepStart)
			stepName := cb.getChildName(orchestration, index)

			if err != nil {
				errorCollector.AddErrorWithOpID(index, err, stepDuration, cb.getChildOperationID(orchestration, index))
				// Continue execution - don't cancel
			} else {
				// Store successful result
				cb.storeOrchestrationResult(result, orchResult, orchestration, stepName)
			}
		}(i, orch)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Return aggregated error if any occurred
	if errorCollector.HasErrors() {
		// Add all errors to result
		for _, opErr := range errorCollector.GetAllErrors() {
			result.AddError(opErr)
		}
		return errorCollector.GetFinalError()
	}

	return nil
}

// getMaxConcurrency determines the maximum number of concurrent operations.
// Uses configuration value if specified, otherwise defaults to a conservative limit.
func (cb *ConcurrentBuilder) getMaxConcurrency(config config.Config) int {
	if config.MaxConcurrency > 0 {
		return config.MaxConcurrency
	}

	// Conservative default: 2x CPU cores, but at least the number of orchestrations
	defaultMax := runtime.NumCPU() * 2
	if defaultMax < len(cb.orchestrations) {
		return len(cb.orchestrations)
	}
	return defaultMax
}

// storeOrchestrationResult stores the result from a child orchestration.
// Handles both merging nested results and storing the main result with proper naming.
func (cb *ConcurrentBuilder) storeOrchestrationResult(result *result.Result, orchResult *result.Result, orch types.Orchestration, stepName string) {
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

// getChildName returns a descriptive name for a child orchestration.
// Uses the orchestration's name if available, otherwise generates a consistent name.
func (cb *ConcurrentBuilder) getChildName(orch types.Orchestration, index int) string {
	if name := orch.GetName(); name != "" {
		return name
	}

	// Generate a consistent name using the index directly
	// This ensures step numbers match the position in the sequence
	return fmt.Sprintf("step-%d", index)
}

// getChildOperationID generates a unique operation ID for a child orchestration.
// Combines concurrent name with child information for traceability.
func (cb *ConcurrentBuilder) getChildOperationID(orch types.Orchestration, index int) string {
	concurrentID := cb.getOperationID()
	childName := cb.getChildName(orch, index)
	return fmt.Sprintf("%s.%s", concurrentID, childName)
}

// getOperationID generates a unique operation ID for traceability.
// Uses the concurrent name if available, otherwise generates a default ID.
func (cb *ConcurrentBuilder) getOperationID() string {
	return cb.GetOperationID(cb)
}

// GetChildAt returns the child orchestration at the specified index.
// This method provides access to individual child orchestrations for introspection,
// debugging, and dynamic orchestration management.
//
// Parameters:
//   - index: Zero-based index of the child orchestration to retrieve
//
// Returns:
//   - types.Orchestration: The child orchestration at the specified index
//   - error: Error if index is out of bounds
//
// Example:
//
//	conc := Concurrent(
//	    Task(func() (string, error) { return "task1", nil }).Named("first-task"),
//	    Task(func() (int, error) { return 42, nil }).Named("second-task"),
//	    Task(func() (bool, error) { return true, nil }).Named("third-task"),
//	)
//
//	// Access the second child (index 1)
//	child, err := conc.GetChildAt(1)
//	if err != nil {
//	    log.Printf("Error accessing child: %v", err)
//	} else {
//	    log.Printf("Child name: %s", child.GetName()) // "second-task"
//	}
func (cb *ConcurrentBuilder) GetChildAt(index int) (types.Orchestration, error) {
	if index < 0 || index >= len(cb.orchestrations) {
		return nil, fmt.Errorf("index %d out of bounds: concurrent has %d orchestrations (valid range: 0-%d)",
			index, len(cb.orchestrations), len(cb.orchestrations)-1)
	}
	return cb.orchestrations[index], nil
}

// GetChildCount returns the total number of child orchestrations in this concurrent.
// This method provides the count of orchestrations that will be executed concurrently.
//
// Returns:
//   - int: The number of child orchestrations
//
// Example:
//
//	conc := Concurrent(task1, task2, task3)
//	count := conc.GetChildCount() // Returns 3
func (cb *ConcurrentBuilder) GetChildCount() int {
	return len(cb.orchestrations)
}

// GetChildren returns a copy of all child orchestrations.
// This method provides access to all child orchestrations for introspection and analysis.
// Returns a copy to prevent external modification of the internal orchestration slice.
//
// Returns:
//   - []types.Orchestration: Copy of all child orchestrations
//
// Example:
//
//	conc := Concurrent(task1, task2, task3)
//	children := conc.GetChildren()
//	for i, child := range children {
//	    log.Printf("Child %d: %s (status: %v)", i, child.GetName(), child.GetStatus())
//	}
func (cb *ConcurrentBuilder) GetChildren() []types.Orchestration {
	// Return a copy to prevent external modification
	children := make([]types.Orchestration, len(cb.orchestrations))
	copy(children, cb.orchestrations)
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
//   - types.Orchestration: The found orchestration (nil if not found)
//
// Example:
//
//	conc := Concurrent(
//	    Task(func() (string, error) { return "data", nil }).Named("fetch-data"),
//	    Task(func() (string, error) { return "processed", nil }).Named("process-data"),
//	)
//
//	index, child := conc.FindChildByName("process-data")
//	if child != nil {
//	    log.Printf("Found 'process-data' at index %d", index) // index = 1
//	} else {
//	    log.Println("Child not found")
//	}
func (cb *ConcurrentBuilder) FindChildByName(name string) (int, types.Orchestration) {
	for i, orch := range cb.orchestrations {
		if cb.getChildName(orch, i) == name {
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
//	conc := Concurrent(
//	    Task(func() (string, error) { return "data", nil }).Named("fetch-data"),
//	    Task(func() (string, error) { return "processed", nil }), // unnamed
//	    Task(func() (string, error) { return "saved", nil }).Named("save-data"),
//	)
//
//	names := conc.GetChildNames()
//	// Returns: ["fetch-data", "step-1", "save-data"]
func (cb *ConcurrentBuilder) GetChildNames() []string {
	names := make([]string, len(cb.orchestrations))
	for i, orch := range cb.orchestrations {
		names[i] = cb.getChildName(orch, i)
	}
	return names
}

// =============================================================================
// PathResolver Interface Implementation
// =============================================================================

// =============================================================================
// Path Resolution Methods - Delegated to Base with Container-Specific Logic
// =============================================================================

// GetCurrentPath returns the current orchestration's full hierarchical path.
// Delegates to base implementation.
func (cb *ConcurrentBuilder) GetCurrentPath() string {
	return cb.BaseOrchestrationBuilder.GetCurrentPath(cb)
}

// GetByPath finds an orchestration by its hierarchical path using dynamic resolution.
// This method allows finding any orchestration in the tree without maintaining a centralized map.
//
// Parameters:
//   - path: Hierarchical path (e.g., "main-pipeline.auth-flow.validate-user")
//
// Returns:
//   - types.Orchestration: Found orchestration (nil if not found)
//   - error: Error if path is invalid or orchestration not found
//
// Example:
//
//	// Find a deeply nested task
//	task, err := concurrent.GetByPath("main-pipeline.auth-flow.validate-user")
//	if err != nil {
//	    log.Printf("Task not found: %v", err)
//	} else {
//	    log.Printf("Found task: %s", task.GetName())
//	}
func (cb *ConcurrentBuilder) GetByPath(path string) (types.Orchestration, error) {
	// Handle self-reference
	if path == cb.GetCurrentPath() {
		return cb, nil
	}
	return cb.pathResolver.GetByPath(path)
}

// ListAllPaths returns all available paths in the orchestration subtree.
// This method performs a depth-first traversal to collect all paths dynamically.
//
// Returns:
//   - []string: All paths in the subtree
//
// Example:
//
//	paths := concurrent.ListAllPaths()
//	for _, path := range paths {
//	    log.Printf("Available path: %s", path)
//	}
func (cb *ConcurrentBuilder) ListAllPaths() []string {
	return cb.pathResolver.ListAllPaths()
}

// FindByName searches for orchestrations by name across the entire subtree.
// This method can return multiple matches if the same name appears at different levels.
//
// Parameters:
//   - name: Name to search for
//
// Returns:
//   - []types.PathMatch: All matching orchestrations with their path information
//
// Example:
//
//	matches := concurrent.FindByName("validate-user")
//	for _, match := range matches {
//	    log.Printf("Found '%s' at path: %s (depth: %d)", name, match.Path, match.Depth)
//	}
func (cb *ConcurrentBuilder) FindByName(name string) []types.PathMatch {
	matches := cb.pathResolver.FindByName(name)

	// Check if current orchestration matches
	if cb.GetName() == name {
		currentMatch := types.PathMatch{
			Path:          cb.GetCurrentPath(),
			Orchestration: cb,
			Depth:         cb.namer.GetContext().GetDepth(),
			Parent:        cb.namer.GetContext().GetParentPath(),
			Type:          "concurrent",
		}
		matches = append([]types.PathMatch{currentMatch}, matches...)
	}

	return matches
}

// =============================================================================
// Advanced Path Query Methods
// =============================================================================

// Query returns a PathQuery instance for advanced path-based queries.
// Delegates to base implementation.
func (cb *ConcurrentBuilder) Query() *types.PathQuery {
	return cb.BaseOrchestrationBuilder.Query(cb)
}

// GetOrchestrationTree returns a tree representation of the orchestration hierarchy.
// This method provides a structured view of the entire orchestration tree.
//
// Returns:
//   - *types.OrchestrationTree: Tree representation
//
// Example:
//
//	tree := concurrent.GetOrchestrationTree()
//	tree.Print() // Prints the tree structure
func (cb *ConcurrentBuilder) GetOrchestrationTree() *types.OrchestrationTree {
	tree := &types.OrchestrationTree{
		Name:          cb.GetName(),
		Path:          cb.GetCurrentPath(),
		Type:          "concurrent",
		Depth:         cb.namer.GetContext().GetDepth(),
		Parent:        nil, // Will be set by parent when building tree
		Orchestration: cb,
		Children:      make([]*types.OrchestrationTree, len(cb.orchestrations)),
	}

	// Build child trees
	for i, child := range cb.orchestrations {
		childTree := child.GetOrchestrationTree()
		childTree.Parent = tree
		tree.Children[i] = childTree
	}

	return tree
}
