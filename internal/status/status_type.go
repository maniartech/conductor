// Package status provides atomic-based status management system following
// Go concurrency patterns for the orchestrator library.
package status

// Status represents the execution state of an orchestration
type Status uint32

// Status constants for orchestration lifecycle
const (
	NotStarted Status = iota
	Running
	Completed
	Cancelled
	Failed
)

// String returns the string representation of the status
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
	case Failed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// IsTerminal returns true if the status represents a terminal state
func (s Status) IsTerminal() bool {
	return s == Completed || s == Cancelled || s == Failed
}

// IsActive returns true if the status represents an active state
func (s Status) IsActive() bool {
	return s == Running
}
