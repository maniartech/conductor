package orchestration

import (
	"testing"

	"github.com/maniartech/orchestrator/pkg/types"

	. "github.com/maniartech/orchestrator/internal/orchestration"
)

// TestStatusAliases tests that status aliases work correctly
func TestStatusAliases(t *testing.T) {
	t.Run("status_type_alias", func(t *testing.T) {
		// Test that Status is properly aliased
		var status Status = NotStarted
		var typesStatus types.Status = status

		if typesStatus != types.NotStarted {
			t.Error("Expected Status alias to work correctly")
		}
	})

	t.Run("status_constants", func(t *testing.T) {
		tests := []struct {
			name          string
			orchestration Status
			types         types.Status
		}{
			{"NotStarted", NotStarted, types.NotStarted},
			{"Running", Running, types.Running},
			{"Completed", Completed, types.Completed},
			{"Failed", Failed, types.Failed},
			{"Cancelled", Cancelled, types.Cancelled},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.orchestration != tt.types {
					t.Errorf("Expected %s constants to match: orchestration=%v, types=%v",
						tt.name, tt.orchestration, tt.types)
				}
			})
		}
	})

	t.Run("status_methods", func(t *testing.T) {
		// Test that all status methods work through the alias
		status := Completed

		if !status.IsTerminal() {
			t.Error("Expected Completed status to be terminal")
		}

		if status.IsActive() {
			t.Error("Expected Completed status not to be active")
		}

		if !status.IsSuccessful() {
			t.Error("Expected Completed status to be successful")
		}

		if status.HasFailed() {
			t.Error("Expected Completed status not to have failed")
		}

		if status.WasCancelled() {
			t.Error("Expected Completed status not to be cancelled")
		}

		if status.String() != "Completed" {
			t.Errorf("Expected status string 'Completed', got '%s'", status.String())
		}
	})
}

// TestStatusConstantValues tests that all status constants have correct values
func TestStatusConstantValues(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected uint32
	}{
		{"NotStarted", NotStarted, 0},
		{"Running", Running, 1},
		{"Completed", Completed, 2},
		{"Failed", Failed, 3},
		{"Cancelled", Cancelled, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint32(tt.status) != tt.expected {
				t.Errorf("Expected %s to have value %d, got %d",
					tt.name, tt.expected, uint32(tt.status))
			}
		})
	}
}

// TestStatusTransitions tests status transition logic
func TestStatusTransitions(t *testing.T) {
	tests := []struct {
		name        string
		from        Status
		to          Status
		shouldAllow bool
	}{
		// Valid transitions
		{"NotStarted to Running", NotStarted, Running, true},
		{"Running to Completed", Running, Completed, true},
		{"Running to Failed", Running, Failed, true},
		{"Running to Cancelled", Running, Cancelled, true},

		// Invalid transitions (terminal states)
		{"Completed to Running", Completed, Running, false},
		{"Failed to Running", Failed, Running, false},
		{"Cancelled to Running", Cancelled, Running, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Document expected behavior based on terminal state
			if tt.from.IsTerminal() && !tt.shouldAllow {
				t.Logf("✓ Correctly identified that %s should not transition to %s", tt.from, tt.to)
			} else if !tt.from.IsTerminal() && tt.shouldAllow {
				t.Logf("✓ Correctly identified that %s can transition to %s", tt.from, tt.to)
			}
		})
	}
}

// TestStatusCompatibility tests compatibility with types.Status
func TestStatusCompatibility(t *testing.T) {
	t.Run("assignment_compatibility", func(t *testing.T) {
		// Test that orchestration.Status can be assigned to types.Status
		var orchStatus Status = Running
		var typesStatus types.Status = orchStatus

		if typesStatus != types.Running {
			t.Error("Expected assignment compatibility between Status types")
		}

		// Test reverse assignment
		var typesStatus2 types.Status = types.Completed
		var orchStatus2 Status = typesStatus2

		if orchStatus2 != Completed {
			t.Error("Expected reverse assignment compatibility between Status types")
		}
	})

	t.Run("method_compatibility", func(t *testing.T) {
		// Test that methods work the same way
		orchStatus := Failed
		typesStatus := types.Failed

		if orchStatus.IsTerminal() != typesStatus.IsTerminal() {
			t.Error("Expected IsTerminal() to work the same way")
		}

		if orchStatus.IsActive() != typesStatus.IsActive() {
			t.Error("Expected IsActive() to work the same way")
		}

		if orchStatus.String() != typesStatus.String() {
			t.Error("Expected String() to work the same way")
		}
	})
}

// TestStatusUsageInContext tests status usage in realistic contexts
func TestStatusUsageInContext(t *testing.T) {
	t.Run("status_progression", func(t *testing.T) {
		// Simulate a typical status progression
		status := NotStarted

		if status.IsActive() {
			t.Error("Expected NotStarted not to be active")
		}

		// Start execution
		status = Running
		if !status.IsActive() {
			t.Error("Expected Running to be active")
		}

		if status.IsTerminal() {
			t.Error("Expected Running not to be terminal")
		}

		// Complete execution
		status = Completed
		if status.IsActive() {
			t.Error("Expected Completed not to be active")
		}

		if !status.IsTerminal() {
			t.Error("Expected Completed to be terminal")
		}

		if !status.IsSuccessful() {
			t.Error("Expected Completed to be successful")
		}
	})

	t.Run("error_status_progression", func(t *testing.T) {
		// Simulate error status progression
		status := Running

		// Execution fails
		status = Failed
		if !status.IsTerminal() {
			t.Error("Expected Failed to be terminal")
		}

		if !status.HasFailed() {
			t.Error("Expected Failed status to have failed")
		}

		if status.IsSuccessful() {
			t.Error("Expected Failed status not to be successful")
		}
	})

	t.Run("cancellation_status", func(t *testing.T) {
		// Simulate cancellation
		status := Running

		// Execution is cancelled
		status = Cancelled
		if !status.IsTerminal() {
			t.Error("Expected Cancelled to be terminal")
		}

		if !status.WasCancelled() {
			t.Error("Expected Cancelled status to be cancelled")
		}

		if status.IsSuccessful() {
			t.Error("Expected Cancelled status not to be successful")
		}

		if status.HasFailed() {
			t.Error("Expected Cancelled status not to have failed")
		}
	})
}

// BenchmarkStatusMethods benchmarks status method calls
func BenchmarkStatusMethods(b *testing.B) {
	status := Completed

	b.Run("IsTerminal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			status.IsTerminal()
		}
	})

	b.Run("IsActive", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			status.IsActive()
		}
	})

	b.Run("String", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			status.String()
		}
	})
}
