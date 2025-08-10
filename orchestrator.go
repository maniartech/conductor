// Package orchestrator provides a military-grade, high-performance goroutine orchestration library
// with zero-allocation execution, comprehensive error handling, and complex nested orchestration support.
//
// The orchestrator allows you to compose complex workflows using a declarative syntax with
// Tasks, Sequential, and Concurrent orchestrations that can be nested to any depth.
//
// Example:
//
//	result, err := orchestrator.Setup(
//	    orchestrator.Sequential(
//	        orchestrator.Task(prepareInfra).Named("infra"),
//	        orchestrator.Concurrent(
//	            orchestrator.Task(processA).Named("process-a"),
//	            orchestrator.Task(processB).Named("process-b"),
//	        ).Named("processing"),
//	        orchestrator.Task(cleanup).Named("cleanup"),
//	    ).Named("main-workflow"),
//	).Await()
package orchestrator

import (
	"context"

	"github.com/maniartech/orchestrator/internal/conditional"
	"github.com/maniartech/orchestrator/internal/config"
	orchContext "github.com/maniartech/orchestrator/internal/context"
	"github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/internal/result"
	"github.com/maniartech/orchestrator/internal/task"
)

// Task creates a new task orchestration with the provided function.
// The function must return a value of type T and an error.
// This is the primary building block for creating individual tasks.
//
// Example:
//
//	// String task
//	stringTask := orchestrator.Task(func() (string, error) {
//	    return "result", nil
//	})
//
//	// Integer task with error
//	intTask := orchestrator.Task(func() (int, error) {
//	    return 42, someError
//	})
//
//	// Custom type task
//	userTask := orchestrator.Task(func() (User, error) {
//	    return User{ID: 123, Name: "John"}, nil
//	})
func Task[T any](fn func() (T, error)) orchestration.Orchestration {
	return task.Task(fn)
}

// Sequential creates a sequential orchestration that executes the provided orchestrations
// one after another in the specified order. If any orchestration fails and the error
// strategy is FailFast, the remaining orchestrations are not executed.
//
// Example:
//
//	sequential := orchestrator.Sequential(
//	    orchestrator.Task(step1).Named("step-1"),
//	    orchestrator.Task(step2).Named("step-2"),
//	    orchestrator.Task(step3).Named("step-3"),
//	).Named("sequential-workflow")
func Sequential(orchestrations ...orchestration.Orchestration) orchestration.Orchestration {
	// TODO: Implement SequentialBuilder in task 4.1
	// For now, return a placeholder that will be implemented in the next task
	panic("Sequential orchestration not yet implemented - will be completed in task 4.1")
}

// Concurrent creates a concurrent orchestration that executes the provided orchestrations
// simultaneously in separate goroutines. The orchestration completes when all
// orchestrations have finished (successfully or with errors).
//
// Example:
//
//	concurrent := orchestrator.Concurrent(
//	    orchestrator.Task(taskA).Named("task-a"),
//	    orchestrator.Task(taskB).Named("task-b"),
//	    orchestrator.Task(taskC).Named("task-c"),
//	).Named("concurrent-workflow")
func Concurrent(orchestrations ...orchestration.Orchestration) orchestration.Orchestration {
	// TODO: Implement ConcurrentBuilder in task 5.1
	// For now, return a placeholder that will be implemented in the next task
	panic("Concurrent orchestration not yet implemented - will be completed in task 5.1")
}

// Conditional creates a conditional orchestration that evaluates a condition function
// and executes either the ifTrue or ifFalse orchestration based on the result.
// The condition function has access to the orchestration context for making decisions
// based on runtime state and can return an error if evaluation fails.
//
// Example:
//
//	conditional := orchestrator.Conditional(
//	    func(ctx orchContext.Context) (bool, error) {
//	        authenticated, ok := ctx.Get("user_authenticated").(bool)
//	        if !ok {
//	            return false, errors.New("authentication status not available")
//	        }
//	        return authenticated, nil
//	    },
//	    orchestrator.Task(fetchUserData).Named("fetch-data"),
//	    orchestrator.Task(redirectToLogin).Named("redirect-login"),
//	).Named("auth-check")
func Conditional(condition func(orchContext.Context) (bool, error), ifTrue, ifFalse orchestration.Orchestration) orchestration.Orchestration {
	return conditional.Conditional(condition, ifTrue, ifFalse)
}

// Workflow represents a complete orchestration workflow that can be executed.
// It provides methods for configuration and execution.
type Workflow struct {
	orchestration orchestration.Orchestration
	config        config.Config
}

// Setup creates a new workflow with the provided orchestration.
// This is the entry point for executing orchestrations.
//
// Example:
//
//	workflow := orchestrator.Setup(
//	    orchestrator.Task(myFunction).Named("my-task"),
//	)
//	result, err := workflow.Await()
func Setup(orchestration orchestration.Orchestration) *Workflow {
	return &Workflow{
		orchestration: orchestration,
		config:        config.DefaultConfig(),
	}
}

// With applies configuration to the workflow.
// Configuration is inherited hierarchically with local overrides.
//
// Example:
//
//	workflow := orchestrator.Setup(myOrchestration).
//	    With(config.Config{
//	        Timeout: 60 * time.Second,
//	        MaxConcurrency: 10,
//	        ErrorStrategy: errors.CollectAll,
//	    })
func (w *Workflow) With(cfg config.Config) *Workflow {
	w.config = cfg
	return w
}

// Await executes the workflow and waits for completion.
// Returns the result of the orchestration or an error if execution failed.
//
// Example:
//
//	result, err := workflow.Await()
//	if err != nil {
//	    log.Printf("Workflow failed: %v", err)
//	    return
//	}
//
//	// Access results by name
//	value := result.Get("task-name")
func (w *Workflow) Await() (*result.Result, error) {
	ctx := context.Background()
	if w.config.Context != nil {
		ctx = w.config.Context
	}

	return w.orchestration.Execute(ctx, w.config)
}

// AwaitWithContext executes the workflow with the provided context and waits for completion.
// The context can be used for cancellation and timeout control.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	result, err := workflow.AwaitWithContext(ctx)
func (w *Workflow) AwaitWithContext(ctx context.Context) (*result.Result, error) {
	return w.orchestration.Execute(ctx, w.config)
}

// GetStatus returns the current status of the workflow orchestration.
// This method is thread-safe and can be called concurrently.
//
// Example:
//
//	status := workflow.GetStatus()
//	fmt.Printf("Workflow status: %s\n", status) // NotStarted, Running, Completed, or Cancelled
func (w *Workflow) GetStatus() orchestration.Status {
	return w.orchestration.GetStatus()
}

// GetName returns the name of the workflow orchestration.
// Returns empty string if no name was set.
func (w *Workflow) GetName() string {
	return w.orchestration.GetName()
}

// GetConfig returns the workflow's configuration.
func (w *Workflow) GetConfig() config.Config {
	return w.config
}

// Re-export commonly used types and constants for convenience
type (
	// Status represents the execution status of an orchestration
	Status = orchestration.Status

	// Config represents orchestration configuration
	Config = config.Config

	// ErrorStrategy represents error handling strategy
	ErrorStrategy = errors.ErrorStrategy

	// Result represents the result of orchestration execution
	Result = result.Result
)

// Re-export status constants
const (
	NotStarted = orchestration.NotStarted
	Running    = orchestration.Running
	Completed  = orchestration.Completed
	Cancelled  = orchestration.Cancelled
)

// Re-export error strategy constants
const (
	FailFast   = errors.FailFast
	CollectAll = errors.CollectAll
)

// DefaultConfig returns the default configuration for orchestrations.
func DefaultConfig() Config {
	return config.DefaultConfig()
}

// Development Status:
// ✅ Task execution engine (Task 3.2) - COMPLETED
//    - Zero-allocation atomic status management
//    - Comprehensive panic recovery with stack traces
//    - Context cancellation and timeout support
//    - Thread-safe operations with race detection
//    - Generic type support with fluent API
//    - Enhanced Orchestration interface with GetStatus(), GetName(), GetConfig()
//
// 🚧 Sequential orchestration (Task 4.1) - PENDING
//    - Sequential execution with fail-fast and collect-all error strategies
//    - Hierarchical configuration inheritance
//    - Rich error metadata collection
//
// 🚧 Concurrent orchestration (Task 5.1) - PENDING
//    - Concurrent execution with goroutine management
//    - Atomic error collection and synchronization
//    - Load balancing and resource management
//
// ✅ Conditional orchestration (Task 6.1) - COMPLETED
//    - Context-based condition evaluation with comprehensive error handling
//    - Branch selection and execution logic with proper resource cleanup
//    - Configuration inheritance to selected branch
//    - Panic recovery for condition evaluation
//
// 🚧 Workflow management (Task 7.1) - PENDING
//    - Complete workflow execution engine
//    - Result aggregation and error handling
