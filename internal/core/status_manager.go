// Package core provides the fundamental types and interfaces for the
// orchestrator library following Go best practices and KISS principles.
package core

import (
	"github.com/maniartech/orchestrator/internal/status"
)

// StatusManager provides atomic status management for orchestrations
type StatusManager struct {
	*status.Manager
}

// NewStatusManager creates a new status manager
func NewStatusManager() *StatusManager {
	return &StatusManager{
		Manager: status.NewManager(),
	}
}
