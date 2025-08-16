package orchestrator

import (
	"testing"
	"time"

	. "github.com/maniartech/orchestrator"
)

// TestAdvancedDeadlockScenarios tests more complex scenarios that might cause deadlocks
func TestAdvancedDeadlockScenarios(t *testing.T) {
	t.Run("sequential_placeholder_panic", func(t *testing.T) {
		// This should panic since Sequential is not implemented
		defer func() {
			if r := recover(); r != nil {
				t.Logf("✅ Sequential correctly panics: %v", r)
			} else {
				t.Error("❌ Expected Sequential to panic")
			}
		}()

		task1 := Task(func() (string, error) {
			return "task1", nil
		}).Named("task1")

		task2 := Task(func() (string, error) {
			return "task2", nil
		}).Named("task2")

		// This should panic
		Sequential(task1, task2)
	})

	t.Run("concurrent_placeholder_panic", func(t *testing.T) {
		// This should panic since Concurrent is not implemented
		defer func() {
			if r := recover(); r != nil {
				t.Logf("✅ Concurrent correctly panics: %v", r)
			} else {
				t.Error("❌ Expected Concurrent to panic")
			}
		}()

		task1 := Task(func() (string, error) {
			return "task1", nil
		}).Named("task1")

		task2 := Task(func() (string, error) {
			return "task2", nil
		}).Named("task2")

		// This should panic
		Concurrent(task1, task2)
	})

	t.Run("multiple_workflows_concurrent", func(t *testing.T) {
		const numWorkflows = 10
		done := make(chan struct{})

		go func() {
			defer close(done)

			results := make(chan *Result, numWorkflows)
			errors := make(chan error, numWorkflows)

			for i := 0; i < numWorkflows; i++ {
				go func(id int) {
					task := Task(func() (string, error) {
						time.Sleep(10 * time.Millisecond)
						return "result", nil
					}).Named("concurrent-task")

					workflow := Setup(task)
					result, err := workflow.ExecuteBlocking()

					if err != nil {
						errors <- err
					} else {
						results <- result
					}
				}(i)
			}

			// Collect results
			successCount := 0
			errorCount := 0

			for i := 0; i < numWorkflows; i++ {
				select {
				case <-results:
					successCount++
				case <-errors:
					errorCount++
				case <-time.After(1 * time.Second):
					t.Errorf("❌ Workflow %d timed out", i)
					return
				}
			}

			t.Logf("✅ Completed %d workflows successfully, %d with errors", successCount, errorCount)
		}()

		select {
		case <-done:
			t.Log("✅ Multiple workflows test completed")
		case <-time.After(10 * time.Second):
			t.Fatal("❌ DEADLOCK DETECTED: Multiple workflows test timed out")
		}
	})

	t.Run("callback_heavy_load", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "heavy-load-result", nil
		}).Named("heavy-load-task")

		workflow := Setup(task)

		// Add many callbacks
		for i := 0; i < 100; i++ {
			workflow.OnProgress(func(progress Progress) {
				// Simulate some work in callback
				time.Sleep(1 * time.Microsecond)
			})

			workflow.OnStatusChange(func(oldStatus, newStatus Status) {
				// Simulate some work in callback
				time.Sleep(1 * time.Microsecond)
			})
		}

		done := make(chan struct{})
		var result *Result
		var err error

		go func() {
			defer close(done)
			result, err = workflow.ExecuteBlocking()
		}()

		select {
		case <-done:
			if err != nil {
				t.Errorf("Workflow failed: %v", err)
			}
			if result == nil {
				t.Error("Expected result")
			}
			t.Log("✅ Heavy callback load test completed")
		case <-time.After(10 * time.Second):
			t.Fatal("❌ DEADLOCK DETECTED: Heavy callback load test timed out")
		}
	})

	t.Run("rapid_workflow_creation", func(t *testing.T) {
		done := make(chan struct{})

		go func() {
			defer close(done)

			for i := 0; i < 1000; i++ {
				task := Task(func() (string, error) {
					return "rapid-result", nil
				}).Named("rapid-task")

				workflow := Setup(task)
				result, err := workflow.ExecuteBlocking()

				if err != nil {
					t.Errorf("Workflow %d failed: %v", i, err)
					return
				}

				if result == nil {
					t.Errorf("Workflow %d returned nil result", i)
					return
				}
			}

			t.Log("✅ Created and executed 1000 workflows successfully")
		}()

		select {
		case <-done:
			t.Log("✅ Rapid workflow creation test completed")
		case <-time.After(30 * time.Second):
			t.Fatal("❌ DEADLOCK DETECTED: Rapid workflow creation test timed out")
		}
	})
}
