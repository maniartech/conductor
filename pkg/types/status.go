package types

// Status represents the execution state of an orchestration.
// Status values are designed for atomic operations and high-performance status checking.
//
// The status follows a well-defined lifecycle:
//   - NotStarted: Initial state when orchestration is created
//   - Running: Orchestration is currently executing
//   - Completed: Orchestration finished successfully
//   - Failed: Orchestration finished with an error
//   - Cancelled: Orchestration was cancelled or timed out
//
// Status transitions are atomic and thread-safe. Once an orchestration reaches
// a terminal state (Completed, Failed, Cancelled), it cannot transition to another state.
//
// Performance characteristics:
//   - Zero-allocation status reads using atomic.Load
//   - Zero-allocation status updates using atomic.Store
//   - Lock-free concurrent access from multiple goroutines
//   - Constant-time status checks and comparisons
//
// Status operations are zero-allocation and use atomic primitives for maximum performance.
// Status reads and updates have minimal overhead suitable for high-frequency operations.
type Status uint32

const (
	// NotStarted indicates the orchestration has been created but not yet executed.
	// This is the initial state for all orchestrations.
	NotStarted Status = iota

	// Running indicates the orchestration is currently executing.
	// The orchestration has started but has not yet completed.
	Running

	// Completed indicates the orchestration finished successfully.
	// This is a terminal state - no further transitions are possible.
	Completed

	// Failed indicates the orchestration finished with an error.
	// This is a terminal state - no further transitions are possible.
	Failed

	// Cancelled indicates the orchestration was cancelled or timed out.
	// This is a terminal state - no further transitions are possible.
	Cancelled
)

// String returns a human-readable representation of the status.
// This method is useful for logging and debugging.
//
// Example:
//
//	status := Running
//	fmt.Printf("Current status: %s", status.String()) // Output: "Current status: Running"
func (s Status) String() string {
	switch s {
	case NotStarted:
		return "NotStarted"
	case Running:
		return "Running"
	case Completed:
		return "Completed"
	case Failed:
		return "Failed"
	case Cancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// IsTerminal returns true if the status represents a terminal state.
// Terminal states are final states that cannot transition to other states.
//
// Terminal states: Completed, Failed, Cancelled
// Non-terminal states: NotStarted, Running
//
// Example:
//
//	if status.IsTerminal() {
//	    fmt.Println("Orchestration has finished")
//	}
func (s Status) IsTerminal() bool {
	return s == Completed || s == Failed || s == Cancelled
}

// IsActive returns true if the orchestration is currently active (running).
// This is a convenience method equivalent to checking if status == Running.
//
// Example:
//
//	if status.IsActive() {
//	    fmt.Println("Orchestration is currently running")
//	}
func (s Status) IsActive() bool {
	return s == Running
}

// IsSuccessful returns true if the orchestration completed successfully.
// This is a convenience method equivalent to checking if status == Completed.
//
// Example:
//
//	if status.IsSuccessful() {
//	    fmt.Println("Orchestration completed successfully")
//	}
func (s Status) IsSuccessful() bool {
	return s == Completed
}

// HasFailed returns true if the orchestration failed.
// This is a convenience method equivalent to checking if status == Failed.
//
// Example:
//
//	if status.HasFailed() {
//	    fmt.Println("Orchestration failed")
//	}
func (s Status) HasFailed() bool {
	return s == Failed
}

// WasCancelled returns true if the orchestration was cancelled.
// This is a convenience method equivalent to checking if status == Cancelled.
//
// Example:
//
//	if status.WasCancelled() {
//	    fmt.Println("Orchestration was cancelled")
//	}
func (s Status) WasCancelled() bool {
	return s == Cancelled
}
