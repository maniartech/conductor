package orchestrator

type Status uint8

// Orchestration status
const (
	// Not Started
	NotStarted Status = iota

	// Pending
	Pending

	// Finished
	Finished

	// Error
	Error
)

func (s Status) String() string {
	switch s {
	case NotStarted:
		return "Not Started"
	case Pending:
		return "Pending"
	case Finished:
		return "Finished"
	case Error:
		return "Error"
	default:
		return "Unknown"
	}
}
