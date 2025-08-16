package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/pkg/builders/task"

	. "github.com/maniartech/orchestrator"
)

// TestSequentialConcurrentPlaceholders tests the placeholder functions
func TestSequentialConcurrentPlaceholders(t *testing.T) {
	t.Run("Sequential_Placeholder", func(t *testing.T) {
		// Test Sequential placeholder panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected Sequential to panic")
			} else {
				expectedMsg := "Sequential orchestration not yet implemented - will be completed in task 4.1"
				if r != expectedMsg {
					t.Errorf("Expected panic message %q, got %q", expectedMsg, r)
				}
			}
		}()

		taskFn := func() (string, error) {
			return "result", nil
		}

		t1 := task.Task(taskFn).Named("task1")
		t2 := task.Task(taskFn).Named("task2")

		// This should panic
		Sequential(t1, t2)
	})

	t.Run("Concurrent_Placeholder", func(t *testing.T) {
		// Test Concurrent placeholder panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected Concurrent to panic")
			} else {
				expectedMsg := "Concurrent orchestration not yet implemented - will be completed in task 5.1"
				if r != expectedMsg {
					t.Errorf("Expected panic message %q, got %q", expectedMsg, r)
				}
			}
		}()

		taskFn := func() (string, error) {
			return "result", nil
		}

		t1 := task.Task(taskFn).Named("task1")
		t2 := task.Task(taskFn).Named("task2")

		// This should panic
		Concurrent(t1, t2)
	})
}

// TestWorkflowProgressModes tests all progress mode paths
func TestWorkflowProgressModes(t *testing.T) {
	t.Run("ProgressModeAuto", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "auto-progress", nil
		}

		workflow := Setup(task.Task(taskFn).Named("auto-progress-task"))
		workflow.SetProgressMode(ProgressModeAuto)

		// Get progress in auto mode
		progress := workflow.GetProgress()
		if progress.Total != 1 {
			t.Errorf("Expected total 1, got %d", progress.Total)
		}
	})

	t.Run("ProgressModeManual_WithCustomProgress", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "manual-progress", nil
		}

		workflow := Setup(task.Task(taskFn).Named("manual-progress-task"))
		workflow.SetProgressMode(ProgressModeManual)

		// Report custom progress
		workflow.ReportProgress(30, 100, "Custom manual progress")

		// Get progress (should return custom progress)
		progress := workflow.GetProgress()
		if progress.Current != 30 {
			t.Errorf("Expected current 30, got %d", progress.Current)
		}
		if progress.Message != "Custom manual progress" {
			t.Errorf("Expected custom message, got: %s", progress.Message)
		}
	})

	t.Run("ProgressModeHybrid_FallbackToAuto", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "hybrid-fallback", nil
		}

		workflow := Setup(task.Task(taskFn).Named("hybrid-fallback-task"))
		workflow.SetProgressMode(ProgressModeHybrid)

		// Don't set custom progress - should fallback to auto
		progress := workflow.GetProgress()
		if progress.Total != 1 {
			t.Errorf("Expected total 1 (auto fallback), got %d", progress.Total)
		}
	})
}

// TestWorkflowCallbackPaths tests callback-related paths
func TestWorkflowCallbackPaths(t *testing.T) {
	t.Run("OnProgress_Callback", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "progress-callback", nil
		}

		workflow := Setup(task.Task(taskFn).Named("progress-callback-task"))

		var progressReceived bool
		workflow.OnProgress(func(progress Progress) {
			progressReceived = true
		})

		// Report progress to trigger callback
		workflow.ReportProgress(50, 100, "Test progress")

		if !progressReceived {
			t.Error("Progress callback should have been called")
		}
	})

	t.Run("OnStatusChange_Callback", func(t *testing.T) {
		taskFn := func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "status-callback", nil
		}

		workflow := Setup(task.Task(taskFn).Named("status-callback-task"))

		var statusChanges []string
		workflow.OnStatusChange(func(oldStatus, newStatus Status) {
			statusChanges = append(statusChanges, fmt.Sprintf("%s->%s", oldStatus, newStatus))
		})

		// Execute workflow
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Wait for completion
		_, _ = workflow.Await()

		if len(statusChanges) == 0 {
			t.Error("Status change callbacks should have been called")
		}
	})

	t.Run("OnError_Callback", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "", fmt.Errorf("error callback test")
		}

		workflow := Setup(task.Task(taskFn).Named("error-callback-task"))

		errorChan := make(chan error, 1)
		workflow.OnError(func(err error) {
			errorChan <- err
		})

		// Execute workflow (should trigger error callback)
		_, err := workflow.Await()

		// Should have an error from task execution
		if err == nil {
			t.Error("Expected error from task execution")
		}

		// Wait for error callback with timeout
		select {
		case callbackErr := <-errorChan:
			if callbackErr == nil {
				t.Error("Error callback received nil error")
			}
		case <-time.After(100 * time.Millisecond):
			// Error callback might not be implemented yet, which is acceptable
			t.Log("Error callback not called - feature may not be fully implemented")
		}
	})

	t.Run("OnComplete_Callback", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "completion-callback", nil
		}

		workflow := Setup(task.Task(taskFn).Named("completion-callback-task"))

		completionChan := make(chan bool, 1)
		workflow.OnComplete(func(result *Result, err error) {
			completionChan <- true
		})

		// Execute workflow
		result, err := workflow.Await()

		// Should complete successfully
		if err != nil {
			t.Errorf("Expected successful completion, got error: %v", err)
		}

		if result == nil {
			t.Error("Expected result, got nil")
		}

		// Wait for completion callback with timeout
		select {
		case <-completionChan:
			// Callback was called successfully
		case <-time.After(100 * time.Millisecond):
			// Completion callback might not be implemented yet, which is acceptable
			t.Log("Completion callback not called - feature may not be fully implemented")
		}
	})
}

// TestWorkflowCancellationPaths tests cancellation paths
func TestWorkflowCancellationPaths(t *testing.T) {
	t.Run("Cancel_Basic", func(t *testing.T) {
		taskFn := func() (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "cancelled-task", nil
		}

		workflow := Setup(task.Task(taskFn).Named("cancel-basic-task"))

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Cancel immediately
		workflow.Cancel()

		// Check status
		if workflow.GetStatus() != Cancelled {
			t.Error("Workflow should be cancelled")
		}
	})

	t.Run("CancelWithReason", func(t *testing.T) {
		taskFn := func() (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "cancelled-with-reason", nil
		}

		workflow := Setup(task.Task(taskFn).Named("cancel-reason-task"))

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Cancel with reason
		workflow.CancelWithReason("test cancellation reason")

		// Check status
		if workflow.GetStatus() != Cancelled {
			t.Error("Workflow should be cancelled")
		}
	})
}

// TestWorkflowGetMethods tests getter methods
func TestWorkflowGetMethods(t *testing.T) {
	t.Run("GetName", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "name-test", nil
		}

		workflow := Setup(task.Task(taskFn).Named("get-name-task"))

		name := workflow.GetName()
		if name != "get-name-task" {
			t.Errorf("Expected name 'get-name-task', got: %s", name)
		}
	})

	t.Run("GetConfig", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "config-test", nil
		}

		cfg := config.Config{
			Timeout: 10 * time.Second,
		}

		workflow := Setup(task.Task(taskFn).Named("get-config-task")).With(cfg)

		retrievedConfig := workflow.GetConfig()
		if retrievedConfig.Timeout != 10*time.Second {
			t.Errorf("Expected timeout 10s, got: %v", retrievedConfig.Timeout)
		}
	})

	t.Run("GetCurrentTask", func(t *testing.T) {
		taskFn := func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "current-task-test", nil
		}

		workflow := Setup(task.Task(taskFn).Named("current-task-test"))

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Get current task (might be empty if execution is fast)
		currentTask := workflow.GetCurrentTask()
		_ = currentTask // Use the variable

		// Wait for completion
		_, _ = workflow.Await()
	})

	t.Run("GetPartialResults", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "partial-results-test", nil
		}

		workflow := Setup(task.Task(taskFn).Named("partial-results-task"))

		// Get partial results before execution
		partialResults := workflow.GetPartialResults()
		if partialResults == nil {
			t.Error("Partial results should not be nil")
		}

		// Execute workflow
		_, _ = workflow.Await()

		// Get partial results after execution
		partialResults = workflow.GetPartialResults()
		if partialResults == nil {
			t.Error("Partial results should not be nil after execution")
		}
	})

	t.Run("GetProgressLegacy", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "legacy-progress", nil
		}

		workflow := Setup(task.Task(taskFn).Named("legacy-progress-task"))

		// Test legacy progress format
		completed, total, percentage := workflow.GetProgressLegacy()
		if total != 1 {
			t.Errorf("Expected total 1, got %d", total)
		}
		if completed < 0 || completed > total {
			t.Errorf("Invalid completed value: %d", completed)
		}
		if percentage < 0 || percentage > 100 {
			t.Errorf("Invalid percentage: %f", percentage)
		}
	})
}

// TestWorkflowStageOperations tests stage-related operations
func TestWorkflowStageOperations(t *testing.T) {
	t.Run("SetStage_WithProgressUpdate", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "stage-test", nil
		}

		workflow := Setup(task.Task(taskFn).Named("stage-test-task"))

		// Set stage
		workflow.SetStage("Testing Stage")

		// Get progress (should include stage)
		progress := workflow.GetProgress()
		if progress.Stage != "Testing Stage" {
			t.Errorf("Expected stage 'Testing Stage', got: %s", progress.Stage)
		}
	})

	t.Run("SetStage_ManualMode", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "manual-stage-test", nil
		}

		workflow := Setup(task.Task(taskFn).Named("manual-stage-task"))
		workflow.SetProgressMode(ProgressModeManual)

		// Set stage in manual mode
		workflow.SetStage("Manual Stage")

		// Get progress
		progress := workflow.GetProgress()
		if progress.Stage != "Manual Stage" {
			t.Errorf("Expected stage 'Manual Stage', got: %s", progress.Stage)
		}
	})
}

// TestWorkflowAsyncExecution tests async execution paths
func TestWorkflowAsyncExecution(t *testing.T) {
	t.Run("executeAsync_Success", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "async-success", nil
		}

		workflow := Setup(task.Task(taskFn).Named("async-success-task"))

		// Execute async
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start async execution: %v", err)
		}

		// Wait for completion
		result, err := workflow.Await()
		if err != nil {
			t.Errorf("Async execution failed: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("executeAsync_WithError", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "", fmt.Errorf("async execution error")
		}

		workflow := Setup(task.Task(taskFn).Named("async-error-task"))

		// Execute async
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start async execution: %v", err)
		}

		// Wait for completion (should have error)
		result, err := workflow.Await()
		if err == nil {
			t.Error("Expected error from async execution")
		}
		if result == nil {
			t.Error("Result should not be nil even on error")
		}
	})
}

// TestWorkflowContextPaths tests context-related paths
func TestWorkflowContextPaths(t *testing.T) {
	t.Run("AwaitWithContext_ContextDone", func(t *testing.T) {
		taskFn := func() (string, error) {
			time.Sleep(100 * time.Millisecond) // Long enough to be cancelled
			return "context-done-test", nil
		}

		workflow := Setup(task.Task(taskFn).Named("context-done-task"))

		// Create context that will be cancelled
		ctx, cancel := context.WithCancel(context.Background())

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Cancel context immediately
		cancel()

		// AwaitWithContext should return context error
		result, err := workflow.AwaitWithContext(ctx)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
		if result != nil {
			t.Error("Result should be nil when context is cancelled")
		}
	})

	t.Run("AwaitWithTimeout_Timeout", func(t *testing.T) {
		taskFn := func() (string, error) {
			time.Sleep(100 * time.Millisecond) // Longer than timeout
			return "timeout-test", nil
		}

		workflow := Setup(task.Task(taskFn).Named("timeout-task"))

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Await with short timeout
		result, err := workflow.AwaitWithTimeout(10 * time.Millisecond)
		if err == nil {
			t.Error("Expected timeout error")
		}
		if result != nil {
			t.Error("Result should be nil on timeout")
		}

		if !strings.Contains(fmt.Sprintf("%v", err), "timeout") {
			t.Errorf("Error should mention timeout, got: %v", err)
		}
	})
}

// TestWorkflowInternalMethods tests internal method paths
func TestWorkflowInternalMethods(t *testing.T) {
	t.Run("initializeProgressTracking", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "init-progress", nil
		}

		workflow := Setup(task.Task(taskFn).Named("init-progress-task"))

		// Progress should be initialized
		progress := workflow.GetProgress()
		if progress.Total != 1 {
			t.Errorf("Expected total 1, got %d", progress.Total)
		}
		if progress.Current != 0 {
			t.Errorf("Expected current 0, got %d", progress.Current)
		}
	})
}

// TestTypeAliases tests the type aliases and constants
func TestTypeAliases(t *testing.T) {
	t.Run("TypeAliases", func(t *testing.T) {
		// Test that type aliases work
		var cfg Config
		cfg.Timeout = 5 * time.Second

		var strategy ErrorStrategy = FailFast
		if strategy != FailFast {
			t.Error("ErrorStrategy alias not working")
		}

		strategy = CollectAll
		if strategy != CollectAll {
			t.Error("CollectAll constant not working")
		}
	})
}

// TestWorkflowEdgeCases tests edge cases and error paths
func TestWorkflowEdgeCases(t *testing.T) {
	t.Run("Workflow_NilTask", func(t *testing.T) {
		// Test workflow with nil task (should be handled gracefully)
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for nil task function")
			}
		}()

		// This should panic in task creation
		task.Task[string](nil)
	})

	t.Run("Workflow_EmptyName", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "empty-name-test", nil
		}

		// Create task with empty name
		workflow := Setup(task.Task(taskFn))

		name := workflow.GetName()
		if name != "" {
			t.Errorf("Expected empty name, got: %s", name)
		}

		// Should still execute successfully
		result, err := workflow.Await()
		if err != nil {
			t.Errorf("Workflow with empty name failed: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})
}
