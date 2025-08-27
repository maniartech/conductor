package orchestrator

import (
	"context"
	"testing"
	"time"

	. "github.com/maniartech/orchestrator"
)

// TestDeadlockReproduction attempts to reproduce the deadlock
func TestDeadlockReproduction(t *testing.T) {
	t.Run("simple_task_execution", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "test-result", nil
		}).Named("test-task")

		workflow := Setup(task)

		// Set a timeout to prevent hanging
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
			t.Log("✅ Test completed successfully")
		case <-time.After(5 * time.Second):
			t.Fatal("❌ DEADLOCK DETECTED: Test timed out after 5 seconds")
		}
	})

	t.Run("task_with_callbacks", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "callback-result", nil
		}).Named("callback-task")

		workflow := Setup(task).
			OnProgress(func(progress Progress) {
				t.Logf("Progress: %d/%d", progress.Current, progress.Total)
			}).
			OnStatusChange(func(oldStatus, newStatus Status) {
				t.Logf("Status: %s -> %s", oldStatus, newStatus)
			}).
			OnComplete(func(result *Result, err error) {
				t.Log("Completion callback called")
			})

		// Set a timeout to prevent hanging
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
			t.Log("✅ Callback test completed successfully")
		case <-time.After(5 * time.Second):
			t.Fatal("❌ DEADLOCK DETECTED: Callback test timed out after 5 seconds")
		}
	})

	t.Run("async_execution", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "async-result", nil
		}).Named("async-task")

		workflow := Setup(task)

		// Set a timeout to prevent hanging
		done := make(chan struct{})
		var result *Result
		var err error

		go func() {
			defer close(done)
			if execErr := workflow.Execute(); execErr != nil {
				t.Errorf("Failed to start workflow: %v", execErr)
				return
			}
			result, err = workflow.Await()
		}()

		select {
		case <-done:
			if err != nil {
				t.Errorf("Workflow failed: %v", err)
			}
			if result == nil {
				t.Error("Expected result")
			}
			t.Log("✅ Async test completed successfully")
		case <-time.After(5 * time.Second):
			t.Fatal("❌ DEADLOCK DETECTED: Async test timed out after 5 seconds")
		}
	})

	t.Run("context_cancellation", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "should-not-complete", nil
		}).Named("cancel-task")

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		workflow := Setup(task)

		// Set a timeout to prevent hanging
		done := make(chan struct{})
		var result *Result
		var err error

		go func() {
			defer close(done)
			result, err = workflow.AwaitWithContext(ctx)
		}()

		select {
		case <-done:
			// Should get a timeout error
			if err == nil {
				t.Error("Expected timeout error")
			}
			if result != nil {
				t.Log("Got result despite cancellation")
			}
			t.Log("✅ Cancellation test completed successfully")
		case <-time.After(5 * time.Second):
			t.Fatal("❌ DEADLOCK DETECTED: Cancellation test timed out after 5 seconds")
		}
	})
}
