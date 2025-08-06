// Package errors provides error handling types and strategies for the orchestrator library.
// It includes ErrorStrategy for controlling error propagation and OperationError
// for detailed error reporting with context and stack traces.
package errors

// ErrorStrategy defines how errors should be handled during orchestration.
// It controls whether execution stops on the first error or continues to collect all errors.
//
// Example:
//
//	config := Config{ErrorStrategy: CollectAll}
//	// This will execute all operations even if some fail
type ErrorStrategy int

const (
	// FailFast stops execution immediately on first error.
	// Remaining operations are cancelled to minimize resource usage.
	FailFast ErrorStrategy = iota

	// CollectAll continues execution and collects all errors.
	// All operations run to completion regardless of individual failures.
	CollectAll
)

// String returns the string representation of ErrorStrategy
func (e ErrorStrategy) String() string {
	switch e {
	case FailFast:
		return "FailFast"
	case CollectAll:
		return "CollectAll"
	default:
		return "Unknown"
	}
}
