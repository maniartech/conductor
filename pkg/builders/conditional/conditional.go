// Package conditional provides conditional orchestration for the orchestrator library.
// It includes the ConditionalBuilder for creating and executing conditional workflows
// with context-based condition evaluation, comprehensive error handling, and proper resource cleanup.
//
// # Conditional Execution
//
// Conditional orchestrations evaluate a condition function and execute either the ifTrue
// or ifFalse orchestration based on the result. The condition function has access to the
// orchestration context for making decisions based on runtime state:
//
//   - Context-based condition evaluation with access to shared state
//   - Branch selection based on condition result
//   - Proper resource cleanup for unused branches
//   - Error handling for condition evaluation failures
//   - Configuration inheritance to selected branch
//
// # Error Handling
//
// Conditional orchestrations provide comprehensive error handling:
//
//   - Condition evaluation errors are captured and reported
//   - Branch execution errors are propagated normally
//   - Unused branches are not executed (resource efficient)
//   - Error boundaries are respected within the selected branch
//
// # Configuration Inheritance
//
// Conditional orchestrations support hierarchical configuration:
//
//	parent := Config{Timeout: 30*time.Second, ErrorStrategy: FailFast}
//	conditional := Conditional(condition, ifTrue, ifFalse).With(Config{Timeout: 60*time.Second})
//	// Selected branch inherits the conditional's configuration
//
// # Thread Safety
//
// All conditional operations are thread-safe using atomic operations for status management.
// Conditional orchestrations can be safely accessed from multiple goroutines concurrently.
//
// # Performance Characteristics
//
// The conditional execution engine is designed for efficiency:
//   - Only the selected branch is executed (no wasted resources)
//   - Zero-allocation status operations using atomic primitives
//   - Efficient condition evaluation with context access
//   - Minimal memory overhead with proper cleanup
//   - Lock-free status management
//
// # Usage Examples
//
//	// Basic conditional execution
//	result, err := Conditional(
//	    func(ctx Context) bool { return ctx.Get("user_authenticated").(bool) },
//	    Task(fetchUserData),
//	    Task(redirectToLogin),
//	).Execute(ctx, config)
//
//	// Named conditional with configuration
//	result, err := Conditional(
//	    func(ctx Context) bool { return ctx.Get("cache_enabled").(bool) },
//	    Task(fetchFromCache),
//	    Task(fetchFromDatabase),
//	).Named("data-source-selector").
//	With(Config{Timeout: 30*time.Second}).
//	Execute(ctx, config)
//
//	// Nested conditional orchestrations
//	result, err := Conditional(
//	    func(ctx Context) bool { return ctx.Get("environment") == "production" },
//	    Conditional(prodCondition, prodTaskA, prodTaskB),
//	    Conditional(devCondition, devTaskA, devTaskB),
//	).Execute(ctx, config)
package conditional

import (
	"context"
	"fmt"
	"time"

	orchContext "github.com/maniartech/orchestrator/internal/context"
	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/result"
	"github.com/maniartech/orchestrator/types"
)

// ConditionalBuilder provides a fluent API for creating and configuring conditional orchestrations.
// It evaluates a condition function and executes either the ifTrue or ifFalse orchestration
// based on the result, with comprehensive error handling and configuration inheritance.
//
// Example:
//
//	conditional := Conditional(
//	    func(ctx Context) bool { return ctx.Get("user_role") == "admin" },
//	    Task(adminOperation),
//	    Task(userOperation),
//	).Named("role-based-operation").
//	With(config.Config{Timeout: 30*time.Second})
type ConditionalBuilder struct {
	*orchestration.BaseOrchestrationBuilder
	condition     func(orchContext.Context) (bool, error)
	ifTrue        types.Orchestration
	ifFalse       types.Orchestration
	namer         *types.HierarchicalNamer        // Hierarchical naming system
	parentContext *types.NamingContext            // Parent naming context for nested orchestrations
	pathResolver  *orchestration.PathResolverBase // Path-based orchestration resolution
}

// Conditional creates a new ConditionalBuilder with the provided condition and orchestrations.
// The condition function is evaluated with access to the orchestration context, and based on
// the result, either ifTrue or ifFalse orchestration is executed.
//
// Parameters:
//   - condition: Function that evaluates to true or false based on context, may return an error
//   - ifTrue: Orchestration to execute if condition returns true
//   - ifFalse: Orchestration to execute if condition returns false
//
// Returns:
//   - *ConditionalBuilder: A new conditional builder instance
//
// Panics:
//   - If condition function is nil
//   - If ifTrue orchestration is nil
//   - If ifFalse orchestration is nil
//
// Example:
//
//	// Simple conditional execution
//	cond := Conditional(
//	    func(ctx Context) (bool, error) {
//	        authenticated, ok := ctx.Get("authenticated").(bool)
//	        if !ok {
//	            return false, errors.New("authentication status not found")
//	        }
//	        return authenticated, nil
//	    },
//	    Task(func() (string, error) { return "Welcome!", nil }),
//	    Task(func() (string, error) { return "Please login", nil }),
//	)
//
//	// Conditional with complex logic and error handling
//	cond := Conditional(
//	    func(ctx Context) (bool, error) {
//	        userRole, ok := ctx.Get("user_role").(string)
//	        if !ok {
//	            return false, errors.New("user role not found in context")
//	        }
//	        return userRole == "admin" || userRole == "moderator", nil
//	    },
//	    Sequential(validateAdmin, executeAdminTask),
//	    Task(executeUserTask),
//	)
func Conditional(condition func(orchContext.Context) (bool, error), ifTrue, ifFalse types.Orchestration) *ConditionalBuilder {
	if condition == nil {
		panic("condition function cannot be nil")
	}
	if ifTrue == nil {
		panic("ifTrue orchestration cannot be nil")
	}
	if ifFalse == nil {
		panic("ifFalse orchestration cannot be nil")
	}

	cb := &ConditionalBuilder{
		BaseOrchestrationBuilder: orchestration.NewBaseOrchestrationBuilder("conditional"),
		condition:                condition,
		ifTrue:                   ifTrue,
		ifFalse:                  ifFalse,
	}

	// Initialize hierarchical naming system
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

// initializeNaming initializes the hierarchical naming system for this conditional builder.
// This method sets up the naming context and hierarchical namer.
func (cb *ConditionalBuilder) initializeNaming() {
	// Initialize with default naming (will be updated if Named() is called)
	cb.namer = types.NewHierarchicalNamer(cb.parentContext, cb.GetName(), "conditional", 0)
}

// SetParentContext sets the parent naming context for nested orchestrations.
// This method is used internally when this conditional is used as a child orchestration.
func (cb *ConditionalBuilder) SetParentContext(parentContext *types.NamingContext, index int) {
	cb.parentContext = parentContext
	cb.namer = types.NewHierarchicalNamer(parentContext, cb.GetName(), "conditional", index)
}

// Named sets a name for the conditional orchestration for observability and debugging.
// The name appears in logs and error messages to help identify which orchestration failed.
// Returns the same ConditionalBuilder instance for method chaining.
//
// Example:
//
//	cond := Conditional(condition, ifTrue, ifFalse).Named("user-access-control")
func (cb *ConditionalBuilder) Named(name string) types.Orchestration {
	cb.SetName(name)
	// Refresh the naming system with the new name
	cb.initializeNaming()
	return cb
}

// With applies configuration to the conditional orchestration.
// Configuration is inherited hierarchically with local overrides.
// The selected branch inherits this configuration unless it specifies its own.
// Returns the same ConditionalBuilder instance for method chaining.
//
// Example:
//
//	cond := Conditional(condition, ifTrue, ifFalse).
//	    With(config.Config{
//	        Timeout: 60*time.Second,
//	        ErrorStrategy: errors.CollectAll,
//	    })
func (cb *ConditionalBuilder) With(config config.Config) types.Orchestration {
	cb.SetConfig(config)
	return cb
}

// ErrorBoundary sets error handling strategy for this conditional orchestration.
// This controls how errors propagate within the selected branch.
// Returns the same ConditionalBuilder instance for method chaining.
//
// Note: The error boundary applies to the selected branch execution, not to the
// condition evaluation itself. Condition evaluation errors are always propagated.
//
// Example:
//
//	cond := Conditional(condition, ifTrue, ifFalse).ErrorBoundary(errors.CollectAll)
func (cb *ConditionalBuilder) ErrorBoundary(strategy errors.ErrorStrategy) types.Orchestration {
	cb.SetErrorBoundary(strategy)
	return cb
}

// Execute runs the conditional orchestration with the provided context and configuration.
// This method implements the core conditional execution logic with comprehensive error handling,
// configuration inheritance, and proper resource management.
//
// The execution process:
// 1. Applies configuration inheritance and error boundary settings
// 2. Creates orchestration context for condition evaluation
// 3. Evaluates the condition function with comprehensive error handling
// 4. Selects and executes the appropriate branch (ifTrue or ifFalse)
// 5. Returns results from the selected branch with proper error propagation
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - config: Base configuration to inherit from
//
// Returns:
//   - *result.Result: Results from the selected branch
//   - error: Condition evaluation error or branch execution error
//
// Example:
//
//	result, err := conditional.Execute(ctx, config)
//	if err != nil {
//	    log.Printf("Conditional execution failed: %v", err)
//	}
//
//	// Access results from the selected branch
//	branchResult := result.Get("selected-branch")
func (cb *ConditionalBuilder) Execute(ctx context.Context, config config.Config) (*result.Result, error) {
	// Validate execution preconditions (handled by base)
	if err := cb.ValidateExecutionPreconditions(); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Apply configuration inheritance (handled by base)
	finalConfig := cb.ApplyConfigurationInheritance(config)

	// Create execution context with timeout
	execCtx := ctx
	if finalConfig.Context != nil && finalConfig.Context != context.Background() {
		execCtx = finalConfig.Context
	}

	if finalConfig.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(execCtx, finalConfig.Timeout)
		defer cancel()
	}

	// Create result container
	result := result.NewResult()

	// Execute conditional logic
	executionError := cb.executeConditional(execCtx, finalConfig, result)

	// Complete execution (handled by base)
	cb.CompleteExecution(execCtx, executionError)

	// Add execution metadata if there were errors
	if executionError != nil && len(result.Errors()) == 0 {
		// Add a conditional-level error if no branch errors were collected
		result.AddError(errors.OperationError{
			Error:     executionError,
			Index:     -1, // Conditional-level error
			Duration:  time.Since(startTime),
			Timestamp: startTime,
			OpID:      cb.GetOperationID(cb),
		})
	}

	return result, executionError
}

// executeConditional implements the core conditional execution logic.
// This method handles condition evaluation, branch selection, and execution with proper error handling.
func (cb *ConditionalBuilder) executeConditional(ctx context.Context, config config.Config, result *result.Result) error {
	// Create orchestration context for condition evaluation
	orchCtx := orchContext.NewContext(config)

	// Evaluate condition with comprehensive error handling
	conditionResult, conditionError := cb.safeEvaluateCondition(ctx, orchCtx)
	if conditionError != nil {
		return fmt.Errorf("condition evaluation failed: %w", conditionError)
	}

	// Select branch based on condition result
	var selectedBranch types.Orchestration
	var branchName string
	if conditionResult {
		selectedBranch = cb.ifTrue
		branchName = "if-true"
	} else {
		selectedBranch = cb.ifFalse
		branchName = "if-false"
	}

	// Execute selected branch
	stepStart := time.Now()
	branchResult, branchError := selectedBranch.Execute(ctx, config)
	stepDuration := time.Since(stepStart)

	if branchError != nil {
		// Add branch execution error
		result.AddError(errors.OperationError{
			Error:     branchError,
			Index:     0, // Single branch execution
			Duration:  stepDuration,
			Timestamp: stepStart,
			OpID:      cb.getBranchOperationID(selectedBranch, branchName),
		})
		return branchError
	}

	// Store successful result
	if branchResult != nil {
		// Merge the branch result to preserve any nested results
		result.Merge(branchResult)

		// Store the main result using the branch name for direct access
		var taskResult any
		if selectedBranch.GetName() != "" {
			taskResult = branchResult.Get(selectedBranch.GetName())
		}
		if taskResult == nil {
			taskResult = branchResult.Get("task_result")
		}

		if taskResult != nil {
			result.Set(branchName, taskResult)
		}

		// Also store with conditional name if available
		if cb.GetName() != "" {
			result.Set(cb.GetName(), taskResult)
		}
	}

	return nil
}

// safeEvaluateCondition evaluates the condition function with comprehensive error handling.
// This method includes panic recovery and timeout handling for condition evaluation.
func (cb *ConditionalBuilder) safeEvaluateCondition(ctx context.Context, orchCtx orchContext.Context) (bool, error) {
	// Channel for condition evaluation completion
	done := make(chan struct{})
	var conditionResult bool
	var conditionError error

	// Evaluate condition in goroutine with panic recovery
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				conditionResult = false
				conditionError = fmt.Errorf("condition evaluation panic: %v", r)
			}
		}()

		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			conditionResult = false
			conditionError = ctx.Err()
			return
		default:
		}

		// Execute the condition function - now handles both result and error
		conditionResult, conditionError = cb.condition(orchCtx)
	}()

	// Wait for completion or cancellation
	select {
	case <-done:
		// Condition evaluation completed (successfully, with error, or panic)
		return conditionResult, conditionError

	case <-ctx.Done():
		// Context was cancelled or timed out
		return false, ctx.Err()
	}
}

// getBranchOperationID generates a unique operation ID for a branch execution.
// Combines conditional name with branch information for traceability.
func (cb *ConditionalBuilder) getBranchOperationID(branch types.Orchestration, branchName string) string {
	conditionalID := cb.getOperationID()
	return fmt.Sprintf("%s.%s", conditionalID, branchName)
}

// getOperationID generates a unique operation ID for traceability.
// Uses the conditional name if available, otherwise generates a default ID.
func (cb *ConditionalBuilder) getOperationID() string {
	return cb.GetOperationID(cb)
}

// GetChildren returns the child orchestrations (ifTrue and ifFalse branches).
// This method provides access to both branches for introspection and analysis.
//
// Returns:
//   - []types.Orchestration: Array containing ifTrue and ifFalse orchestrations
//
// Example:
//
//	cond := Conditional(condition, taskA, taskB)
//	children := cond.GetChildren()
//	// children[0] is taskA (ifTrue), children[1] is taskB (ifFalse)
func (cb *ConditionalBuilder) GetChildren() []types.Orchestration {
	return []types.Orchestration{cb.ifTrue, cb.ifFalse}
}

// getChildName returns a descriptive name for a child orchestration.
// Uses the orchestration's name if available, otherwise generates a consistent name.
func (cb *ConditionalBuilder) getChildName(orch types.Orchestration, index int) string {
	if name := orch.GetName(); name != "" {
		return name
	}

	// Generate branch-specific names
	if index == 0 {
		return "if-true"
	} else if index == 1 {
		return "if-false"
	}

	// Fallback for unexpected indices
	return fmt.Sprintf("branch-%d", index)
}

// =============================================================================
// Path Resolution Methods - Delegated to Base with Container-Specific Logic
// =============================================================================

// GetCurrentPath returns the current orchestration's full hierarchical path.
// Delegates to base implementation.
func (cb *ConditionalBuilder) GetCurrentPath() string {
	return cb.BaseOrchestrationBuilder.GetCurrentPath(cb)
}

// GetByPath finds an orchestration by its hierarchical path using dynamic resolution.
// This method allows finding any orchestration in the tree without maintaining a centralized map.
//
// Parameters:
//   - path: Hierarchical path (e.g., "main-pipeline.user-auth.if-true")
//
// Returns:
//   - types.Orchestration: Found orchestration (nil if not found)
//   - error: Error if path is invalid or orchestration not found
//
// Example:
//
//	// Find a branch in a conditional
//	branch, err := conditional.GetByPath("main-pipeline.user-auth.if-true")
//	if err != nil {
//	    log.Printf("Branch not found: %v", err)
//	} else {
//	    log.Printf("Found branch: %s", branch.GetName())
//	}
func (cb *ConditionalBuilder) GetByPath(path string) (types.Orchestration, error) {
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
//	paths := conditional.ListAllPaths()
//	for _, path := range paths {
//	    log.Printf("Available path: %s", path)
//	}
func (cb *ConditionalBuilder) ListAllPaths() []string {
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
//	matches := conditional.FindByName("validate-user")
//	for _, match := range matches {
//	    log.Printf("Found '%s' at path: %s (depth: %d)", name, match.Path, match.Depth)
//	}
func (cb *ConditionalBuilder) FindByName(name string) []types.PathMatch {
	matches := cb.pathResolver.FindByName(name)

	// Check if current orchestration matches
	if cb.GetName() == name {
		currentMatch := types.PathMatch{
			Path:          cb.GetCurrentPath(),
			Orchestration: cb,
			Depth:         cb.namer.GetContext().GetDepth(),
			Parent:        cb.namer.GetContext().GetParentPath(),
			Type:          "conditional",
		}
		matches = append([]types.PathMatch{currentMatch}, matches...)
	}

	return matches
}

// Query returns a PathQuery instance for advanced path-based queries.
// Delegates to base implementation.
func (cb *ConditionalBuilder) Query() *types.PathQuery {
	return cb.BaseOrchestrationBuilder.Query(cb)
}

// GetOrchestrationTree returns a tree representation of the orchestration hierarchy.
// This method provides a structured view of the conditional and its branches.
//
// Returns:
//   - *types.OrchestrationTree: Tree representation
//
// Example:
//
//	tree := conditional.GetOrchestrationTree()
//	tree.Print() // Prints the tree structure
func (cb *ConditionalBuilder) GetOrchestrationTree() *types.OrchestrationTree {
	tree := &types.OrchestrationTree{
		Name:          cb.GetName(),
		Path:          cb.GetCurrentPath(),
		Type:          "conditional",
		Depth:         cb.namer.GetContext().GetDepth(),
		Parent:        nil, // Will be set by parent when building tree
		Orchestration: cb,
		Children:      make([]*types.OrchestrationTree, 2), // ifTrue and ifFalse
	}

	// Build child trees
	children := cb.GetChildren()
	for i, child := range children {
		childTree := child.GetOrchestrationTree()
		childTree.Parent = tree
		tree.Children[i] = childTree
	}

	return tree
}
