package builders

import (
	"strings"
	"testing"

	"github.com/maniartech/orchestrator/pkg/builders/concurrent"
	"github.com/maniartech/orchestrator/pkg/builders/conditional"
	"github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
)

// TestTaskBuilder_GetOperationID tests the GetOperationID method for TaskBuilder
func TestTaskBuilder_GetOperationID(t *testing.T) {
	t.Run("Task with name", func(t *testing.T) {
		taskBuilder := task.Task(func(ctx orchContext.Context) (string, error) {
			return "test", nil
		}).Named("test-task")

		opID := taskBuilder.GetOperationID()
		if !strings.Contains(opID, "task-test-task") {
			t.Errorf("Expected operation ID to contain 'task-test-task', got: %s", opID)
		}
	})

	t.Run("Task without name", func(t *testing.T) {
		taskBuilder := task.Task(func(ctx orchContext.Context) (string, error) {
			return "test", nil
		})

		opID := taskBuilder.GetOperationID()
		if !strings.HasPrefix(opID, "task-") {
			t.Errorf("Expected operation ID to start with 'task-', got: %s", opID)
		}
		if strings.Contains(opID, "task-test-task") {
			t.Errorf("Expected operation ID to not contain specific name, got: %s", opID)
		}
	})

	t.Run("Task operation ID consistency", func(t *testing.T) {
		taskBuilder := task.Task(func(ctx orchContext.Context) (string, error) {
			return "test", nil
		}).Named("consistent-task")

		opID1 := taskBuilder.GetOperationID()
		opID2 := taskBuilder.GetOperationID()

		if opID1 != opID2 {
			t.Errorf("Expected consistent operation IDs, got: %s != %s", opID1, opID2)
		}
	})
}

// TestSequentialBuilder_GetOperationID tests the GetOperationID method for SequentialBuilder
func TestSequentialBuilder_GetOperationID(t *testing.T) {
	t.Run("Sequential with name", func(t *testing.T) {
		seq := sequential.Sequential(
			task.Task(func(ctx orchContext.Context) (string, error) { return "1", nil }),
			task.Task(func(ctx orchContext.Context) (string, error) { return "2", nil }),
		).Named("test-sequential")

		opID := seq.GetOperationID()
		if !strings.Contains(opID, "sequential-test-sequential") {
			t.Errorf("Expected operation ID to contain 'sequential-test-sequential', got: %s", opID)
		}
	})

	t.Run("Sequential without name", func(t *testing.T) {
		seq := sequential.Sequential(
			task.Task(func(ctx orchContext.Context) (string, error) { return "1", nil }),
			task.Task(func(ctx orchContext.Context) (string, error) { return "2", nil }),
		)

		opID := seq.GetOperationID()
		if !strings.HasPrefix(opID, "sequential-") {
			t.Errorf("Expected operation ID to start with 'sequential-', got: %s", opID)
		}
	})

	t.Run("Sequential operation ID consistency", func(t *testing.T) {
		seq := sequential.Sequential(
			task.Task(func(ctx orchContext.Context) (string, error) { return "1", nil }),
		).Named("consistent-sequential")

		opID1 := seq.GetOperationID()
		opID2 := seq.GetOperationID()

		if opID1 != opID2 {
			t.Errorf("Expected consistent operation IDs, got: %s != %s", opID1, opID2)
		}
	})
}

// TestConcurrentBuilder_GetOperationID tests the GetOperationID method for ConcurrentBuilder
func TestConcurrentBuilder_GetOperationID(t *testing.T) {
	t.Run("Concurrent with name", func(t *testing.T) {
		conc := concurrent.Concurrent(
			task.Task(func(ctx orchContext.Context) (string, error) { return "1", nil }),
			task.Task(func(ctx orchContext.Context) (string, error) { return "2", nil }),
		).Named("test-concurrent")

		opID := conc.GetOperationID()
		if !strings.Contains(opID, "concurrent-test-concurrent") {
			t.Errorf("Expected operation ID to contain 'concurrent-test-concurrent', got: %s", opID)
		}
	})

	t.Run("Concurrent without name", func(t *testing.T) {
		conc := concurrent.Concurrent(
			task.Task(func(ctx orchContext.Context) (string, error) { return "1", nil }),
			task.Task(func(ctx orchContext.Context) (string, error) { return "2", nil }),
		)

		opID := conc.GetOperationID()
		if !strings.HasPrefix(opID, "concurrent-") {
			t.Errorf("Expected operation ID to start with 'concurrent-', got: %s", opID)
		}
	})

	t.Run("Concurrent operation ID consistency", func(t *testing.T) {
		conc := concurrent.Concurrent(
			task.Task(func(ctx orchContext.Context) (string, error) { return "1", nil }),
		).Named("consistent-concurrent")

		opID1 := conc.GetOperationID()
		opID2 := conc.GetOperationID()

		if opID1 != opID2 {
			t.Errorf("Expected consistent operation IDs, got: %s != %s", opID1, opID2)
		}
	})
}

// TestConditionalBuilder_GetOperationID tests the GetOperationID method for ConditionalBuilder
func TestConditionalBuilder_GetOperationID(t *testing.T) {
	t.Run("Conditional with name", func(t *testing.T) {
		cond := conditional.Conditional(
			func(ctx orchContext.Context) (bool, error) { return true, nil },
			task.Task(func(ctx orchContext.Context) (string, error) { return "true", nil }),
			task.Task(func(ctx orchContext.Context) (string, error) { return "false", nil }),
		).Named("test-conditional")

		opID := cond.GetOperationID()
		if !strings.Contains(opID, "conditional-test-conditional") {
			t.Errorf("Expected operation ID to contain 'conditional-test-conditional', got: %s", opID)
		}
	})

	t.Run("Conditional without name", func(t *testing.T) {
		cond := conditional.Conditional(
			func(ctx orchContext.Context) (bool, error) { return true, nil },
			task.Task(func(ctx orchContext.Context) (string, error) { return "true", nil }),
			task.Task(func(ctx orchContext.Context) (string, error) { return "false", nil }),
		)

		opID := cond.GetOperationID()
		if !strings.HasPrefix(opID, "conditional-") {
			t.Errorf("Expected operation ID to start with 'conditional-', got: %s", opID)
		}
	})

	t.Run("Conditional operation ID consistency", func(t *testing.T) {
		cond := conditional.Conditional(
			func(ctx orchContext.Context) (bool, error) { return true, nil },
			task.Task(func(ctx orchContext.Context) (string, error) { return "true", nil }),
			task.Task(func(ctx orchContext.Context) (string, error) { return "false", nil }),
		).Named("consistent-conditional")

		opID1 := cond.GetOperationID()
		opID2 := cond.GetOperationID()

		if opID1 != opID2 {
			t.Errorf("Expected consistent operation IDs, got: %s != %s", opID1, opID2)
		}
	})
}

// TestOperationID_InterfaceCompliance tests that all orchestration types properly implement the interface
func TestOperationID_InterfaceCompliance(t *testing.T) {
	taskOrch := task.Task(func(ctx orchContext.Context) (string, error) {
		return "test", nil
	}).Named("interface-task")

	sequentialOrch := sequential.Sequential(taskOrch).Named("interface-sequential")

	concurrentOrch := concurrent.Concurrent(taskOrch).Named("interface-concurrent")

	conditionalOrch := conditional.Conditional(
		func(ctx orchContext.Context) (bool, error) { return true, nil },
		taskOrch,
		taskOrch,
	).Named("interface-conditional")

	// Test that all orchestrations implement the interface and GetOperationID returns non-empty strings
	orchestrations := []struct {
		name           string
		orch           interface{ GetOperationID() string }
		expectedPrefix string
	}{
		{"Task", taskOrch, "task-"},
		{"Sequential", sequentialOrch, "sequential-"},
		{"Concurrent", concurrentOrch, "concurrent-"},
		{"Conditional", conditionalOrch, "conditional-"},
	}

	for _, test := range orchestrations {
		t.Run(test.name, func(t *testing.T) {
			opID := test.orch.GetOperationID()
			if opID == "" {
				t.Errorf("Expected non-empty operation ID for %s", test.name)
			}
			if !strings.HasPrefix(opID, test.expectedPrefix) {
				t.Errorf("Expected operation ID for %s to start with '%s', got: %s", test.name, test.expectedPrefix, opID)
			}
		})
	}
}

// TestOperationID_Uniqueness tests that different orchestration instances have unique operation IDs
func TestOperationID_Uniqueness(t *testing.T) {
	task1 := task.Task(func(ctx orchContext.Context) (string, error) { return "1", nil })
	task2 := task.Task(func(ctx orchContext.Context) (string, error) { return "2", nil })

	opID1 := task1.GetOperationID()
	opID2 := task2.GetOperationID()

	if opID1 == opID2 {
		t.Errorf("Expected unique operation IDs, got: %s == %s", opID1, opID2)
	}
}

// TestOperationID_NestedOrchestrations tests operation IDs in nested orchestrations
func TestOperationID_NestedOrchestrations(t *testing.T) {
	innerTask := task.Task(func(ctx orchContext.Context) (string, error) {
		return "inner", nil
	}).Named("inner-task")

	outerSequential := sequential.Sequential(innerTask).Named("outer-sequential")

	// Test that each has its own unique operation ID
	innerOpID := innerTask.GetOperationID()
	outerOpID := outerSequential.GetOperationID()

	if innerOpID == outerOpID {
		t.Errorf("Expected different operation IDs for nested orchestrations, got: %s == %s", innerOpID, outerOpID)
	}

	if !strings.Contains(innerOpID, "task-inner-task") {
		t.Errorf("Expected inner task operation ID to contain 'task-inner-task', got: %s", innerOpID)
	}

	if !strings.Contains(outerOpID, "sequential-outer-sequential") {
		t.Errorf("Expected outer sequential operation ID to contain 'sequential-outer-sequential', got: %s", outerOpID)
	}
}

// TestOperationID_ThreadSafety tests that GetOperationID is thread-safe
func TestOperationID_ThreadSafety(t *testing.T) {
	taskOrch := task.Task(func(ctx orchContext.Context) (string, error) {
		return "test", nil
	}).Named("thread-safe-task")

	const goroutineCount = 100
	opIDs := make(chan string, goroutineCount)

	// Launch multiple goroutines calling GetOperationID
	for i := 0; i < goroutineCount; i++ {
		go func() {
			opIDs <- taskOrch.GetOperationID()
		}()
	}

	// Collect all operation IDs
	firstOpID := <-opIDs
	for i := 1; i < goroutineCount; i++ {
		opID := <-opIDs
		if opID != firstOpID {
			t.Errorf("Expected consistent operation ID from concurrent calls, got: %s != %s", firstOpID, opID)
		}
	}
}
