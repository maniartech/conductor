package sequential

import (
	"testing"

	"github.com/maniartech/orchestrator/internal/orchestration"
	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/builders/task"
)

// TestSequential_Constructor verifies constructor panics and basic initialization
func TestSequential_Constructor(t *testing.T) {
	// Valid construction
	seq := Sequential(
		task.Task(func() (string, error) { return "a", nil }),
		task.Task(func() (int, error) { return 1, nil }),
	)
	if seq == nil {
		t.Fatal("expected non-nil sequential builder")
	}
	if seq.GetStatus() != orchestration.NotStarted {
		t.Errorf("expected initial status NotStarted, got %v", seq.GetStatus())
	}
	if seq.GetChildCount() != 2 {
		t.Errorf("expected 2 children, got %d", seq.GetChildCount())
	}

	// Panic: zero orchestrations
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for zero orchestrations")
			}
		}()
		Sequential()
	}()

	// Panic: nil child
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil orchestration")
			}
		}()
		Sequential(task.Task(func() (string, error) { return "x", nil }), nil)
	}()
}
