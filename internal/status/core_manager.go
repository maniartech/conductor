package status

// StatusManager provides atomic status management for orchestrations
type StatusManager struct {
	*Manager
}

// NewStatusManager creates a new status manager
func NewStatusManager() *StatusManager {
	return &StatusManager{
		Manager: NewManager(),
	}
}
