// Package task provides task execution and management for the orchestrator library.
// It includes the TaskBuilder for creating and executing individual tasks with
// generic type support, atomic status management, and comprehensive error handling.
package task

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/internal/result"
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
	fn     func() (T, error)
	name   string
	config *config.Config
	status atomic.Uint32 // Atomic status management
}

// Task creates a new TaskBuilder with the provided function.
// The function must return a value of type T and an error.
// This is the primary constructor for creating individual tasks.
//
// Example:
//
//	// String task
//	stringTask := Task(func() (string, error) {
//	    return "result", nil
//	})
//
//	// Integer task
//	intTask := Task(func() (int, error) {
//	    return 42, nil
//	})
//
//	// Custom type task
//	userTask := Task(func() (User, error) {
//	    return User{ID: 123, Name: "John"}, nil
//	})
func Task[T any](fn func() (T, error)) *TaskBuilder[T] {
	if fn == nil {
		panic("task function cannot be nil")
	}
	return &TaskBuilder[T]{
		fn: fn,
	}
}

// Named sets a name for the task for observability and debugging.
// The name appears in logs and error messages to help identify which task failed.
// Returns the same TaskBuilder instance for method chaining.
//
// Example:
//
//	task := Task(fetchUserData).Named("fetch-user")
func (tb *TaskBuilder[T]) Named(name string) orchestration.Orchestration {
	tb.name = name
	return tb
}

// With applies configuration to the task.
// Configuration is inherited hierarchically with local overrides.
// Returns the same TaskBuilder instance for method chaining.
//
// Example:
//
//	task := Task(longRunningOperation).
//	    With(config.Config{Timeout: 60*time.Second})
func (tb *TaskBuilder[T]) With(config config.Config) orchestration.Orchestration {
	tb.config = &config
	return tb
}

// ErrorBoundary sets error handling strategy for this task.
// This is a convenience method that updates the task's configuration with the specified error strategy.
// Returns the same TaskBuilder instance for method chaining.
//
// Example:
//
//	task := Task(riskyOperation).ErrorBoundary(CollectAll)
func (tb *TaskBuilder[T]) ErrorBoundary(strategy errors.ErrorStrategy) orchestration.Orchestration {
	if tb.config == nil {
		tb.config = &config.Config{ErrorStrategy: strategy}
	} else {
		// Create a new config with the updated error strategy
		newConfig := *tb.config
		newConfig.ErrorStrategy = strategy
		tb.config = &newConfig
	}
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
	// Ensure task can only be executed once
	if !tb.compareAndSwapStatus(TaskNotStarted, TaskRunning) {
		return nil, fmt.Errorf("task already executed or in progress, current status: %v", tb.GetStatus())
	}

	startTime := time.Now()

	// Apply configuration inheritance
	finalConfig := config
	if tb.config != nil {
		finalConfig = tb.config.Inherit(config)
	}

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
	taskResult, taskError := tb.safeExecute(execCtx)
	duration := time.Since(startTime)

	// Update status based on outcome
	if taskError != nil {
		if execCtx.Err() != nil {
			tb.setStatus(TaskCancelled)
		} else {
			tb.setStatus(TaskCompleted)
		}

		result.AddError(errors.OperationError{
			Error:     taskError,
			Index:     0,
			Duration:  duration,
			Timestamp: startTime,
			OpID:      tb.getOperationID(),
			Stack:     tb.captureStack(),
		})
		return result, taskError
	}

	// Task completed successfully
	tb.setStatus(TaskCompleted)

	// Store result with task name or default name
	resultName := tb.name
	if resultName == "" {
		resultName = "task_result"
	}
	result.Set(resultName, taskResult)
	return result, nil
}

// safeExecute runs the task function with comprehensive panic recovery and cancellation support.
// This method implements the core execution logic with proper error handling and resource cleanup.
func (tb *TaskBuilder[T]) safeExecute(ctx context.Context) (T, error) {
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

		// Execute the actual task function
		taskResult, taskError = tb.fn()
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

// getOperationID generates a unique operation ID for traceability.
// Uses the task name if available, otherwise generates a default ID.
func (tb *TaskBuilder[T]) getOperationID() string {
	if tb.name != "" {
		return fmt.Sprintf("task-%s", tb.name)
	}
	return fmt.Sprintf("task-%p", tb)
}

// GetName returns the task name for debugging and observability.
// Returns empty string if no name was set.
func (tb *TaskBuilder[T]) GetName() string {
	return tb.name
}

// GetConfig returns the task's configuration.
// Returns nil if no configuration was set.
func (tb *TaskBuilder[T]) GetConfig() *config.Config {
	return tb.config
}

// GetStatus returns the current task status using atomic operations.
// This method is thread-safe and can be called concurrently.
func (tb *TaskBuilder[T]) GetStatus() TaskStatus {
	return TaskStatus(tb.status.Load())
}

// setStatus atomically sets the task status.
// This is an internal method used during task execution.
func (tb *TaskBuilder[T]) setStatus(status TaskStatus) {
	tb.status.Store(uint32(status))
}

// compareAndSwapStatus atomically compares and swaps the task status.
// Returns true if the swap was successful, false otherwise.
// This ensures thread-safe status transitions.
func (tb *TaskBuilder[T]) compareAndSwapStatus(old, new TaskStatus) bool {
	return tb.status.CompareAndSwap(uint32(old), uint32(new))
}
