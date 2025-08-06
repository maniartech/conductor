package core

import (
	"testing"
)

func TestTaskStatus(t *testing.T) {
	// Test TaskStatus string representation
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskNotStarted, "NotStarted"},
		{TaskRunning, "Running"},
		{TaskCompleted, "Completed"},
		{TaskCancelled, "Cancelled"},
		{TaskStatus(999), "Unknown"},
	}

	for _, test := range tests {
		if got := test.status.String(); got != test.expected {
			t.Errorf("TaskStatus(%d).String() = %q, want %q", test.status, got, test.expected)
		}
	}
}
