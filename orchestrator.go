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
	"time"

	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/internal/status"
	"github.com/maniartech/orchestrator/pkg/builders/concurrent"
	"github.com/maniartech/orchestrator/pkg/builders/conditional"
	"github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
	"github.com/maniartech/orchestrator/pkg/result"
)

// Context is an alias for the orchestrator context type for convenience
type Context = orchContext.Context

// Task creates a new task orchestration with the provided function.
// The function receives the orchestrator context and must return a value of type T and an error.
// This is the primary building block for creating individual tasks.
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
//	stringTask := orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
//	    return "result", nil
//	})
//
//	// Integer task with context usage
//	intTask := orchestrator.Task(func(ctx orchestrator.Context) (int, error) {
//	    userID := ctx.Get("user_id").(int)
//	    return userID * 2, nil
//	})
//
//	// Custom type task
//	userTask := orchestrator.Task(func(ctx orchestrator.Context) (User, error) {
//	    return User{ID: 123, Name: "John"}, nil
//	})
func Task[T any](fn func(Context) (T, error)) orchestration.Orchestration {
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
	return sequential.Sequential(orchestrations...)
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
	return concurrent.Concurrent(orchestrations...)
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

// Progress represents workflow execution progress following industry standards
type Progress struct {
	Current    int64     `json:"current"`
	Total      int64     `json:"total"`
	Percentage float64   `json:"percentage"`
	Message    string    `json:"message,omitempty"`
	Stage      string    `json:"stage,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type Status = status.Status

// // Status represents the execution state of a workflow
// type Status uint32

// // Status constants for workflow lifecycle
// const (
// 	NotStarted Status = iota
// 	Running
// 	Completed
// 	Cancelled
// 	Failed
// )

// // String returns the string representation of the status
// func (s Status) String() string {
// 	switch s {
// 	case NotStarted:
// 		return "NotStarted"
// 	case Running:
// 		return "Running"
// 	case Completed:
// 		return "Completed"
// 	case Cancelled:
// 		return "Cancelled"
// 	case Failed:
// 		return "Failed"
// 	default:
// 		return "Unknown"
// 	}
// }

// // IsTerminal returns true if the status represents a terminal state
// func (s Status) IsTerminal() bool {
// 	return s == Completed || s == Cancelled || s == Failed
// }

// // IsActive returns true if the status represents an active state
// func (s Status) IsActive() bool {
// 	return s == Running
// }

// Callback function types following industry standards
type ProgressCallback func(progress Progress)
type StatusCallback func(oldStatus, newStatus Status)
type ErrorCallback func(err error)
type CompletionCallback func(result *result.Result, err error)

// Progress tracking modes
const (
	ProgressModeAuto   uint32 = iota // Automatic task counting (default)
	ProgressModeManual               // Manual progress reporting only
	ProgressModeHybrid               // Combination of both
)
