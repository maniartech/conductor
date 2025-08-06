package status

import (
	"testing"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{NotStarted, "NotStarted"},
		{Running, "Running"},
		{Completed, "Completed"},
		{Cancelled, "Cancelled"},
		{Failed, "Failed"},
		{Status(999), "Unknown"},
	}

	for _, test := range tests {
		if got := test.status.String(); got != test.expected {
			t.Errorf("Status(%d).String() = %q, want %q", test.status, got, test.expected)
		}
	}
}

func TestStatusIsTerminal(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{NotStarted, false},
		{Running, false},
		{Completed, true},
		{Cancelled, true},
		{Failed, true},
	}

	for _, test := range tests {
		if got := test.status.IsTerminal(); got != test.expected {
			t.Errorf("Status(%v).IsTerminal() = %v, want %v", test.status, got, test.expected)
		}
	}
}

func TestStatusIsActive(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{NotStarted, false},
		{Running, true},
		{Completed, false},
		{Cancelled, false},
		{Failed, false},
	}

	for _, test := range tests {
		if got := test.status.IsActive(); got != test.expected {
			t.Errorf("Status(%v).IsActive() = %v, want %v", test.status, got, test.expected)
		}
	}
}
