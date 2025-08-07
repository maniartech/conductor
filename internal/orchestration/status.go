package orchestration

// Status represents the execution status of an orchestration using atomic operations.
// It provides thread-safe status management for concurrent orchestration execution.
//
// # Status Transitions
//
// Valid status transitions are:
//   - NotStarted → Running (when execution begins)
//   - Running → Completed (when execution finishes)
//   - Running → Cancelled (when execution is cancelled or times out)
//
// Invalid transitions (like Completed → Running) are prevented by atomic compare-and-swap operations.
//
// # Thread Safety
//
// Status operations are thread-safe using atomic operations from sync/atomic.
// Multiple goroutines can safely read and update orchestration status concurrently.
//
// # Performance
//
// Status operations are zero-allocation and use atomic primitives for maximum performance.
// Status reads and updates have minimal overhead suitable for high-frequency operations.
type Status uint32

const (
	// NotStarted indicates the orchestration has not begun execution
	NotStarted Status = iota
	// Running indicates the orchestration is currently executing
	Running
	// Completed indicates the orchestration finished successfully or with an error
	Completed
	// Cancelled indicates the orchestration was cancelled before completion
	Cancelled
)

// String returns the string representation of Status
func (s Status) String() string {
	switch s {
	case NotStarted:
		return "NotStarted"
	case Running:
		return "Running"
	case Completed:
		return "Completed"
	case Cancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}
