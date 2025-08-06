// Package status provides atomic-based status management system following
// Go concurrency patterns for the orchestrator library.
package status

import (
	"sync/atomic"
)

// Manager provides atomic status management with thread-safe operations
type Manager struct {
	status atomic.Uint32
}

// NewManager creates a new status manager initialized to NotStarted
func NewManager() *Manager {
	return &Manager{
		status: atomic.Uint32{},
	}
}

// Get returns the current status
func (m *Manager) Get() Status {
	return Status(m.status.Load())
}

// Set atomically sets the status to the new value
func (m *Manager) Set(newStatus Status) {
	m.status.Store(uint32(newStatus))
}

// CompareAndSwap atomically compares the current status with expected
// and swaps to new status if they match. Returns true if swap occurred.
func (m *Manager) CompareAndSwap(expected, new Status) bool {
	return m.status.CompareAndSwap(uint32(expected), uint32(new))
}

// TryTransition attempts to transition from expected status to new status.
// Returns true if transition was successful, false otherwise.
func (m *Manager) TryTransition(expected, new Status) bool {
	return m.CompareAndSwap(expected, new)
}

// TransitionToRunning attempts to transition from NotStarted to Running
func (m *Manager) TransitionToRunning() bool {
	return m.TryTransition(NotStarted, Running)
}

// TransitionToCompleted attempts to transition from Running to Completed
func (m *Manager) TransitionToCompleted() bool {
	return m.TryTransition(Running, Completed)
}

// TransitionToCancelled attempts to transition to Cancelled from any non-terminal state
func (m *Manager) TransitionToCancelled() bool {
	for {
		current := m.Get()
		if current.IsTerminal() {
			return false // Already in terminal state
		}
		if m.TryTransition(current, Cancelled) {
			return true
		}
		// Retry if status changed between Get() and TryTransition()
	}
}

// TransitionToFailed attempts to transition to Failed from any non-terminal state
func (m *Manager) TransitionToFailed() bool {
	for {
		current := m.Get()
		if current.IsTerminal() {
			return false // Already in terminal state
		}
		if m.TryTransition(current, Failed) {
			return true
		}
		// Retry if status changed between Get() and TryTransition()
	}
}

// IsNotStarted returns true if status is NotStarted
func (m *Manager) IsNotStarted() bool {
	return m.Get() == NotStarted
}

// IsRunning returns true if status is Running
func (m *Manager) IsRunning() bool {
	return m.Get() == Running
}

// IsCompleted returns true if status is Completed
func (m *Manager) IsCompleted() bool {
	return m.Get() == Completed
}

// IsCancelled returns true if status is Cancelled
func (m *Manager) IsCancelled() bool {
	return m.Get() == Cancelled
}

// IsFailed returns true if status is Failed
func (m *Manager) IsFailed() bool {
	return m.Get() == Failed
}

// IsTerminal returns true if status is in a terminal state
func (m *Manager) IsTerminal() bool {
	return m.Get().IsTerminal()
}

// IsActive returns true if status is in an active state
func (m *Manager) IsActive() bool {
	return m.Get().IsActive()
}

// Reset atomically resets the status to NotStarted
func (m *Manager) Reset() {
	m.Set(NotStarted)
}
