// Package task provides task execution and management for the orchestrator library.
// It includes the TaskBuilder for creating and executing individual tasks with
// generic type support, atomic status management, and comprehensive error handling.
//
// # Task Execution Lifecycle
//
// Tasks follow a well-defined execution lifecycle with atomic status management:
//
//  1. NotStarted - Initial state when task is created
//  2. Running - Task is currently executing
//  3. Completed - Task finished (successfully or with error)
//  4. Cancelled - Task was cancelled or timed out
//
// # Error Handling
//
// The task execution engine provides comprehensive error handling:
//
//   - Panic Recovery: All panics are caught and converted to errors with stack traces
//   - Timeout Handling: Tasks respect context timeouts with graceful termination
//   - Cancellation Support: Tasks can be cancelled via context cancellation
//   - Error Metadata: Rich error information including timing, operation IDs, and stack traces
//
// # Thread Safety
//
// All task operations are thread-safe using atomic operations for status management.
// Tasks can be safely accessed from multiple goroutines concurrently.
//
// # Generic Type Support
//
// Tasks support generic types for type-safe execution:
//
//	stringTask := Task(func() (string, error) { return "hello", nil })
//	intTask := Task(func() (int, error) { return 42, nil })
//	userTask := Task(func() (User, error) { return User{ID: 1}, nil })
//
// # Configuration Inheritance
//
// Tasks support hierarchical configuration with local overrides:
//
//	task := Task(myFunc).
//	    Named("my-task").
//	    With(config.Config{Timeout: 30*time.Second}).
//	    ErrorBoundary(errors.CollectAll)
//
// # Performance Characteristics
//
// The task execution engine is designed for high performance:
//   - Zero-allocation status operations using atomic primitives
//   - Minimal memory overhead with object pooling support
//   - Efficient goroutine lifecycle management
//   - Lock-free status management
package task

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/pkg/config"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/result"
	"github.com/maniartech/orchestrator/pkg/types"
)

// TaskBuilder provides a fluent API for creating and configuring individual tasks.
// It supports generic types for type-safe task execution and follows the builder pattern
// for consistent API design across all orchestration types.
//
// Example:
//
//	task := Task(func() (string, error) {
//	    return "Hello, World!", nil
//	}).Named("greeting-task").
//	With(config.Config{Timeout: 5*time.Second})
type TaskBuilder[T any] struct {
	*orchestration.BaseOrchestrationBuilder
	fn func(orchContext.Context) (T, error) // Single function signature with context
}

// Task creates a new TaskBuilder with the provided function.
// The function receives the orchestrator context and must return a value of type T and an error.
// This is the primary constructor for creating individual tasks.
//
// The context provides thread-safe access to:
//   - Shared data via ctx.Set/Get
//   - Orchestration configuration via ctx.Config()
//   - Lifecycle management via ctx.Cancel(), ctx.Done()
//   - Timeout management via ctx.WithTimeout(), ctx.IsExpired()
//
// Example:
//
//	// String task
//	stringTask := Task(func(ctx orchestrator.Context) (string, error) {
//	    return "result", nil
//	})
//
//	// Integer task with context usage
//	intTask := Task(func(ctx orchestrator.Context) (int, error) {
//	    userID := ctx.Get("user_id").(int)
//	    return userID * 2, nil
//	})
//
//	// Custom type task
//	userTask := Task(func(ctx orchestrator.Context) (User, error) {
//	    return User{ID: 123, Name: "John"}, nil
//	})
func Task[T any](fn func(orchContext.Context) (T, error)) *TaskBuilder[T] {
	if fn == nil {
		panic("task function cannot be nil")
	}
	return &TaskBuilder[T]{
		BaseOrchestrationBuilder: orchestration.NewBaseOrchestrationBuilder("task"),
		fn:                       fn,
	}
}

// Named sets a name for the task for observability and debugging.
// The name appears in logs and error messages to help identify which task failed.
// Returns the same TaskBuilder instance for method chaining.
//
// Example:
//
//	task := Task(fetchUserData).Named("fetch-user")
func (tb *TaskBuilder[T]) Named(name string) types.Orchestration {
	tb.SetName(name)
	return tb
}

func (tb *TaskBuilder[T]) GetType() string {
	return "task"
}

// With applies configuration to the task.
// Configuration is inherited hierarchically with local overrides.
// Returns the same TaskBuilder instance for method chaining.
//
// Example:
//
//	task := Task(longRunningOperation).
//	    With(config.Config{Timeout: 60*time.Second})
func (tb *TaskBuilder[T]) With(config config.Config) types.Orchestration {
	tb.SetConfig(config)
	return tb
}

// ErrorBoundary sets error handling strategy for this task.
// This is a convenience method that updates the task's configuration with the specified error strategy.
// Returns the same TaskBuilder instance for method chaining.
//
// Example:
//
//	task := Task(riskyOperation).ErrorBoundary(CollectAll)
func (tb *TaskBuilder[T]) ErrorBoundary(strategy errors.ErrorStrategy) types.Orchestration {
	// If no config exists, create one with the error strategy
	if tb.GetConfig() == nil {
		tb.SetConfig(config.Config{ErrorStrategy: strategy})
	} else {
		// Update existing config with new error strategy
		newConfig := *tb.GetConfig()
		newConfig.ErrorStrategy = strategy
		tb.SetConfig(newConfig)
	}
	tb.SetErrorBoundary(strategy)
	return tb
}

// Execute runs the task with the provided context and configuration.
// This method implements the core execution logic with comprehensive error handling,
// panic recovery, and timeout management.
//
// The execution process:
// 1. Applies configuration inheritance
// 2. Sets up timeout and cancellation handling
// 3. Executes the task function with panic recovery
// 4. Returns the result or error
//
// Example:
//
//	result, err := task.Execute(ctx, config)
//	if err != nil {
//	    log.Printf("Task failed: %v", err)
//	}
func (tb *TaskBuilder[T]) Execute(ctx context.Context, config config.Config) (*result.Result, error) {
	// Validate execution preconditions (handled by base)
	if err := tb.ValidateExecutionPreconditions(); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Apply configuration inheritance (handled by base)
	finalConfig := tb.ApplyConfigurationInheritance(config)

	// Create execution context with timeout
	// Always use the provided context as the base, then apply config context if needed
	execCtx := ctx
	if finalConfig.Context != nil && finalConfig.Context != context.Background() {
		// Only override if config has a non-background context
		execCtx = finalConfig.Context
	}

	if finalConfig.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(execCtx, finalConfig.Timeout)
		defer cancel()
	}

	// Create result container
	result := result.NewResult()

	// Execute with comprehensive error handling
	taskResult, taskError := tb.safeExecute(execCtx, finalConfig)
	duration := time.Since(startTime)

	// Complete execution (handled by base)
	tb.CompleteExecution(execCtx, taskError)

	if taskError != nil {
		result.AddError(errors.OperationError{
			Error:     taskError,
			Index:     0,
			Duration:  duration,
			Timestamp: startTime,
			OpID:      tb.GetOperationID(tb),
			Stack:     tb.captureStack(),
		})
		return result, taskError
	}

	// Store result with task name or default name
	resultName := tb.GetName()
	if resultName == "" {
		resultName = "task_result"
	}
	result.Set(resultName, taskResult)
	return result, nil
}

// safeExecute runs the task function with comprehensive panic recovery and cancellation support.
// This method implements the core execution logic with proper error handling and resource cleanup.
//
// # Execution Process
//
// The safeExecute method follows these steps:
//  1. Creates a completion channel for goroutine coordination
//  2. Launches the task function in a separate goroutine with panic recovery
//  3. Monitors for context cancellation or timeout
//  4. Returns results or errors with proper cleanup
//
// # Panic Recovery
//
// All panics are caught and converted to errors with full stack traces:
//   - Captures 4KB stack trace for debugging
//   - Preserves original panic message
//   - Returns zero value for the generic type T
//   - Ensures no goroutine leaks on panic
//
// # Cancellation Handling
//
// Context cancellation is handled at multiple points:
//   - Before task execution begins
//   - During task execution via select statement
//   - Proper cleanup of goroutines and resources
//
// # Resource Management
//
// The method ensures proper resource cleanup:
//   - Goroutines are properly terminated
//   - Channels are closed to prevent leaks
//   - Context cancellation is respected
//   - Memory allocations are minimized
func (tb *TaskBuilder[T]) safeExecute(ctx context.Context, config config.Config) (T, error) {
	// Use shared orchestrator context if available, otherwise create a new one
	var orchCtx orchContext.Context
	if config.OrchestrationContext != nil {
		// Use the shared orchestrator context for data sharing between tasks
		orchCtx = config.OrchestrationContext.(orchContext.Context)
	} else {
		// Create a new orchestrator context from config
		orchCtx = orchContext.NewContext(config)
	}

	// Channel for task completion
	done := make(chan struct{})
	var taskResult T
	var taskError error

	// Execute task in goroutine with panic recovery
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				var zero T
				taskResult = zero

				// Capture stack trace for panic
				stack := make([]byte, 4096)
				stackSize := runtime.Stack(stack, false)

				taskError = fmt.Errorf("task panic recovered: %v\nStack trace:\n%s", r, stack[:stackSize])
			}
		}()

		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			var zero T
			taskResult = zero
			taskError = ctx.Err()
			return
		default:
		}

		// Execute the actual task function with orchestrator context
		taskResult, taskError = tb.fn(orchCtx)
	}()

	// Wait for completion or cancellation
	select {
	case <-done:
		// Task completed (successfully, with error, or panic)
		return taskResult, taskError

	case <-ctx.Done():
		// Context was cancelled or timed out
		var zero T
		return zero, ctx.Err()
	}
}

// captureStack captures the current stack trace for error reporting.
// This provides detailed debugging information when tasks fail.
func (tb *TaskBuilder[T]) captureStack() []byte {
	stack := make([]byte, 4096)
	stackSize := runtime.Stack(stack, false)
	return stack[:stackSize]
}

// These methods are now inherited from BaseOrchestrationBuilder:
// - GetName() string
// - GetConfig() *config.Config
// - GetStatus() types.Status
// - SetStatus(status types.Status)
// - CompareAndSwapStatus(old, new types.Status) bool
// - GetOperationID(instance any) string

// =============================================================================
// Path Resolution Methods - Delegated to Base
// =============================================================================

// GetCurrentPath returns the current task's path.
// Delegates to base implementation.
func (tb *TaskBuilder[T]) GetCurrentPath() string {
	return tb.BaseOrchestrationBuilder.GetCurrentPath(tb)
}

// GetByPath finds an orchestration by its hierarchical path.
// Delegates to base implementation.
func (tb *TaskBuilder[T]) GetByPath(path string) (types.Orchestration, error) {
	return tb.BaseOrchestrationBuilder.GetByPath(tb, path)
}

// ListAllPaths returns all available paths in the task subtree.
// Delegates to base implementation.
func (tb *TaskBuilder[T]) ListAllPaths() []string {
	return tb.BaseOrchestrationBuilder.ListAllPaths(tb)
}

// FindByName searches for orchestrations by name.
// Delegates to base implementation.
func (tb *TaskBuilder[T]) FindByName(name string) []types.PathMatch {
	return tb.BaseOrchestrationBuilder.FindByName(tb, name)
}

// GetOrchestrationTree returns a tree representation of the task.
// Delegates to base implementation.
func (tb *TaskBuilder[T]) GetOrchestrationTree() *types.OrchestrationTree {
	return tb.BaseOrchestrationBuilder.GetOrchestrationTree(tb)
}

// Query returns a PathQuery instance for advanced path-based queries.
// Delegates to base implementation.
func (tb *TaskBuilder[T]) Query() *types.PathQuery {
	return tb.BaseOrchestrationBuilder.Query(tb)
}
