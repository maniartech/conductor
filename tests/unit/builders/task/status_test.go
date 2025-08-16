package task

import (
	"testing"

	"github.com/maniartech/orchestrator/internal/orchestration"
)

func TestTaskStatus(t *testing.T) {
	// Test Status string representation
	tests := []struct {
		status   orchestration.Status
		expected string
	}{
		{orchestration.NotStarted, "NotStarted"},
		{orchestration.Running, "Running"},
		{orchestration.Completed, "Completed"},
		{orchestration.Cancelled, "Cancelled"},
		{orchestration.Status(999), "Unknown"},
	}

	for _, test := range tests {
		if got := test.status.String(); got != test.expected {
			t.Errorf("Status(%d).String() = %q, want %q", test.status, got, test.expected)
		}
	}
}
