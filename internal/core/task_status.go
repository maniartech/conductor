// Package core provides the fundamental types and interfaces for the
// orchestrator library following Go best practices and KISS principles.
package core

// TaskStatus represents the execution status of a task using atomic operations
type TaskStatus uint32

const (
	// TaskNotStarted indicates the task has not begun execution
	TaskNotStarted TaskStatus = iota
	// TaskRunning indicates the task is currently executing
	TaskRunning
	// TaskCompleted indicates the task finished successfully or with an error
	TaskCompleted
	// TaskCancelled indicates the task was cancelled before completion
	TaskCancelled
)

// String returns the string representation of TaskStatus
func (ts TaskStatus) String() string {
	switch ts {
	case TaskNotStarted:
		return "NotStarted"
	case TaskRunning:
		return "Running"
	case TaskCompleted:
		return "Completed"
	case TaskCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}
