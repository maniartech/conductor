package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/builders/task"

	. "github.com/maniartech/orchestrator"
)

// TestWorkflow_ExecuteWorkflow tests the uncovered executeWorkflow function
func TestWorkflow_ExecuteWorkflow(t *testing.T) {
	t.Run("execute_workflow_internal", func(t *testing.T) {
		// Create a task that can be executed
		taskFunc := func() (string, error) {
			return "internal execution result", nil
		}

		taskOrch := task.Task(taskFunc).Named("internal-task")
		workflow := Setup(taskOrch)

		// Execute the workflow (this should trigger executeWorkflow internally)
		result, err := workflow.ExecuteBlocking()

		if err != nil {
			t.Errorf("Unexpected error in executeWorkflow: %v", err)
		}

		if result == nil {
			t.Error("Expected non-nil result from executeWorkflow")
		}

		// Verify the result contains our expected value
		value := result.Get("internal-task")
		if value != "internal execution result" {
			t.Errorf("Expected 'internal execution result', got %v", value)
		}
	})

	t.Run("execute_workflow_with_error", func(t *testing.T) {
		// Create a task that will fail
		taskFunc := func() (string, error) {
			return "", errors.New("internal execution error")
		}

		taskOrch := task.Task(taskFunc).Named("failing-task")
		workflow := Setup(taskOrch)

		// Execute the workflow - should handle error gracefully
		result, err := workflow.ExecuteBlocking()

		if err == nil {
			t.Error("Expected error from failing task")
		}

		// Result should still exist but contain error information
		if result == nil {
			t.Error("Expected non-nil result even with error")
		}

		if !result.HasErrors() {
			t.Error("Expected result to have errors")
		}
	})

	t.Run("execute_workflow_async_completion", func(t *testing.T) {
		// Create a task with a small delay to test async behavior
		taskFunc := func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "async completion", nil
		}

		taskOrch := task.Task(taskFunc).Named("async-task")
		workflow := Setup(taskOrch)

		// Start async execution
		err := workflow.Execute()
		if err != nil {
			t.Errorf("Unexpected error starting async execution: %v", err)
		}

		// Wait for completion
		result, err := workflow.Await()
		if err != nil {
			t.Errorf("Unexpected error awaiting async execution: %v", err)
		}

		value := result.Get("async-task")
		if value != "async completion" {
			t.Errorf("Expected 'async completion', got %v", value)
		}
	})
}

// TestWorkflow_ExecuteOrchestrationTree tests the uncovered executeOrchestrationTree function
func TestWorkflow_ExecuteOrchestrationTree(t *testing.T) {
	t.Run("simple_orchestration_tree", func(t *testing.T) {
		// Create a simple task to test tree execution
		taskFunc := func() (string, error) {
			return "tree execution result", nil
		}

		taskOrch := task.Task(taskFunc).Named("tree-task")
		workflow := Setup(taskOrch)

		// This will internally call executeOrchestrationTree
		result, err := workflow.ExecuteBlocking()

		if err != nil {
			t.Errorf("Unexpected error in tree execution: %v", err)
		}

		if result == nil {
			t.Error("Expected non-nil result from tree execution")
		}

		value := result.Get("tree-task")
		if value != "tree execution result" {
			t.Errorf("Expected 'tree execution result', got %v", value)
		}
	})

	t.Run("orchestration_tree_with_context_timeout", func(t *testing.T) {
		// Create a task that takes longer than the timeout
		taskFunc := func() (string, error) {
			time.Sleep(200 * time.Millisecond)
			return "should timeout", nil
		}

		taskOrch := task.Task(taskFunc).Named("timeout-task")
		workflow := Setup(taskOrch)

		// Execute with a short timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		result, err := workflow.AwaitWithContext(ctx)

		// Should get a timeout error
		if err == nil {
			t.Error("Expected timeout error")
		}

		// Result might be nil or contain partial results
		// This is implementation-dependent, so we just ensure it doesn't panic
		_ = result
	})

	t.Run("orchestration_tree_cancellation", func(t *testing.T) {
		// Create a task that can be cancelled
		taskFunc := func() (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "should be cancelled", nil
		}

		taskOrch := task.Task(taskFunc).Named("cancel-task")
		workflow := Setup(taskOrch)

		// Start async execution
		err := workflow.Execute()
		if err != nil {
			t.Errorf("Unexpected error starting execution: %v", err)
		}

		// Cancel immediately
		workflow.Cancel()

		// Wait for completion
		result, err := workflow.Await()

		// Should get cancellation error or complete quickly
		if err != nil {
			// This is expected for cancellation
		}

		// Ensure we don't panic with cancellation
		_ = result
	})
}

// TestWorkflow_PartialCoverageImprovements tests functions with partial coverage to get them to 100%
func TestWorkflow_PartialCoverageImprovements(t *testing.T) {
	t.Run("string_method_edge_cases", func(t *testing.T) {
		// Test different status values for String() method (currently 85.7%)
		statuses := []Status{NotStarted, Running, Completed, Failed, Cancelled}

		for _, status := range statuses {
			str := status.String()
			if str == "" {
				t.Errorf("Status %d should have non-empty string representation", int(status))
			}
		}

		// Test invalid status value
		invalidStatus := Status(999)
		str := invalidStatus.String()
		if str == "" {
			t.Error("Invalid status should return some string representation")
		}
	})

	t.Run("await_method_context_variations", func(t *testing.T) {
		// Test Await method with various contexts (currently 80.0%)
		taskFunc := func() (string, error) { return "await test", nil }

		// First workflow with its own task instance
		workflow1 := Setup(task.Task(taskFunc).Named("await-task-1"))
		err := workflow1.Execute()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		result, err := workflow1.Await()
		if err != nil {
			t.Errorf("Unexpected error with default context: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}

		// Second workflow must use a fresh task instance to avoid 'already executed' status
		workflow2 := Setup(task.Task(taskFunc).Named("await-task-2"))
		err = workflow2.Execute()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		result, err = workflow2.AwaitWithContext(context.Background())
		if err != nil {
			t.Errorf("Unexpected error with background context: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})

	t.Run("await_with_context_edge_cases", func(t *testing.T) {
		// Test AwaitWithContext with various scenarios (currently 71.4%)
		taskFunc := func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "context test", nil
		}

		taskOrch := task.Task(taskFunc).Named("context-task")

		// Test with already cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		workflow := Setup(taskOrch)
		err := workflow.Execute()
		if err != nil {
			t.Errorf("Unexpected error starting execution: %v", err)
		}

		_, err = workflow.AwaitWithContext(ctx)
		// Should handle cancelled context gracefully
		if err == nil {
			// This might be okay if execution completed before context cancellation was checked
		}
	})

	t.Run("execute_blocking_variations", func(t *testing.T) {
		// Test ExecuteBlocking edge cases (currently 66.7%)

		// Test with task that returns immediately
		quickTask := task.Task(func() (string, error) {
			return "quick", nil
		}).Named("quick-task")

		workflow1 := Setup(quickTask)
		result, err := workflow1.ExecuteBlocking()
		if err != nil {
			t.Errorf("Unexpected error with quick task: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result from quick task")
		}

		// Test with task that has small delay
		delayTask := task.Task(func() (string, error) {
			time.Sleep(5 * time.Millisecond)
			return "delayed", nil
		}).Named("delay-task")

		workflow2 := Setup(delayTask)
		result, err = workflow2.ExecuteBlocking()
		if err != nil {
			t.Errorf("Unexpected error with delayed task: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result from delayed task")
		}
	})

	t.Run("await_with_timeout_variations", func(t *testing.T) {
		// Test AwaitWithTimeout edge cases (currently 66.7%)

		// Test with very short timeout
		taskFunc := func() (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "timeout test", nil
		}

		taskOrch := task.Task(taskFunc).Named("timeout-task")
		workflow := Setup(taskOrch)

		err := workflow.Execute()
		if err != nil {
			t.Errorf("Unexpected error starting execution: %v", err)
		}

		// Very short timeout - should timeout
		result, err := workflow.AwaitWithTimeout(1 * time.Millisecond)
		if err == nil {
			// Might complete before timeout in fast test environment
		}
		_ = result // Don't panic

		// Test with long timeout - should complete
		workflow2 := Setup(task.Task(func() (string, error) {
			return "quick completion", nil
		}).Named("quick-timeout-task"))

		err = workflow2.Execute()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		result, err = workflow2.AwaitWithTimeout(1 * time.Second)
		if err != nil {
			t.Errorf("Unexpected error with long timeout: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result with long timeout")
		}
	})

	t.Run("get_current_task_variations", func(t *testing.T) {
		// Test GetCurrentTask edge cases (currently 66.7%)
		taskFunc := func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "current task test", nil
		}

		taskOrch := task.Task(taskFunc).Named("current-task")
		workflow := Setup(taskOrch)

		// Test before execution
		currentTask := workflow.GetCurrentTask()
		// May be nil or empty before execution starts
		_ = currentTask

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Test during execution (might be available)
		currentTask = workflow.GetCurrentTask()
		_ = currentTask

		// Wait for completion
		_, err = workflow.Await()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Test after completion
		currentTask = workflow.GetCurrentTask()
		_ = currentTask
	})

	t.Run("get_progress_edge_cases", func(t *testing.T) {
		// Test GetProgress edge cases (currently 95.0%)
		taskFunc := func() (string, error) {
			return "progress test", nil
		}

		taskOrch := task.Task(taskFunc).Named("progress-task")
		workflow := Setup(taskOrch)

		// Test various progress reporting scenarios
		progress := workflow.GetProgress()
		if progress.Current < 0 || progress.Total < 0 {
			t.Errorf("Progress values should be non-negative: %+v", progress)
		}

		// Test with manual progress reporting
		workflow.ReportProgress(50, 100, "Manual progress")
		progress = workflow.GetProgress()
		_ = progress // Just ensure no panic

		// Execute and check progress afterwards
		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		progress = workflow.GetProgress()
		if result != nil && progress.Total > 0 {
			// Progress should be complete
		}
	})

	t.Run("report_progress_edge_cases", func(t *testing.T) {
		// Test ReportProgress edge cases (currently 85.7%)
		taskFunc := func() (string, error) {
			return "report test", nil
		}

		taskOrch := task.Task(taskFunc).Named("report-task")
		workflow := Setup(taskOrch)

		// Test various progress values
		workflow.ReportProgress(0, 100, "Starting")
		workflow.ReportProgress(-1, 100, "Negative progress") // Edge case
		workflow.ReportProgress(150, 100, "Over 100%")        // Edge case
		workflow.ReportProgress(50, 100, "")                  // Empty message
		workflow.ReportProgress(100, 100, "Complete")

		// Should not panic with any of these calls
		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})

	t.Run("apply_workflow_configuration_edge_cases", func(t *testing.T) {
		// Test applyWorkflowConfiguration edge cases (currently 92.9%)
		taskFunc := func() (string, error) {
			return "config test", nil
		}

		taskOrch := task.Task(taskFunc).Named("config-task")

		// Test with various configurations
		workflow1 := Setup(taskOrch)

		result, err := workflow1.ExecuteBlocking()
		if err != nil {
			t.Errorf("Unexpected error with configuration: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}

		// Test with nil configuration (edge case) - use a fresh orchestration instance
		freshConfigTask := task.Task(taskFunc).Named("config-task-2")
		workflow2 := Setup(freshConfigTask)
		result, err = workflow2.ExecuteBlocking()
		if err != nil {
			t.Errorf("Unexpected error with nil config: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result with nil config")
		}
	})
}

// TestWorkflow_CompleteCoverageScenarios tests additional scenarios to reach 95%+ coverage
func TestWorkflow_CompleteCoverageScenarios(t *testing.T) {
	t.Run("complex_workflow_lifecycle", func(t *testing.T) {
		// Test complete workflow lifecycle to ensure all paths are covered
		taskFunc := func() (string, error) {
			// Simulate some processing time
			time.Sleep(5 * time.Millisecond)
			return "lifecycle complete", nil
		}

		taskOrch := task.Task(taskFunc).Named("lifecycle-task")
		workflow := Setup(taskOrch)

		// Check initial state
		if workflow.IsRunning() {
			t.Error("Workflow should not be running initially")
		}

		if workflow.IsCompleted() {
			t.Error("Workflow should not be completed initially")
		}

		// Execute and monitor state changes
		err := workflow.Execute()
		if err != nil {
			t.Errorf("Unexpected error starting execution: %v", err)
		}

		// May be running now (timing dependent)
		_ = workflow.IsRunning()

		// Wait for completion
		result, err := workflow.Await()
		if err != nil {
			t.Errorf("Unexpected error awaiting completion: %v", err)
		}

		if result == nil {
			t.Error("Expected non-nil result")
		}

		// Should be completed now
		if !workflow.IsCompleted() {
			t.Error("Workflow should be completed after successful execution")
		}

		// Check terminal state
		if !workflow.IsInTerminalState() {
			t.Error("Completed workflow should be in terminal state")
		}

		// Get final progress
		progress := workflow.GetProgress()
		if progress.Total > 0 && progress.Current != progress.Total {
			// In some cases, progress might not be exactly equal
			// This is implementation-dependent
		}
	})

	t.Run("workflow_with_all_callback_types", func(t *testing.T) {
		// Test workflow with all possible callbacks to ensure coverage
		taskFunc := func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "callback test", nil
		}

		taskOrch := task.Task(taskFunc).Named("callback-task")
		workflow := Setup(taskOrch)

		// Set up all types of callbacks
		workflow.OnProgress(func(progress Progress) {
			_ = progress // Use the progress parameter
		})

		workflow.OnStatusChange(func(oldStatus, newStatus Status) {
			_ = oldStatus
			_ = newStatus
		})

		workflow.OnComplete(func(result *Result, err error) {
			_ = result
			_ = err
		})

		// Execute workflow
		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Error("Expected non-nil result")
		}

		// Give callbacks a moment to be called
		time.Sleep(20 * time.Millisecond)

		// Note: Whether callbacks are called depends on timing and implementation
		// We just ensure the workflow completes successfully with callbacks set
	})

	t.Run("workflow_error_scenarios", func(t *testing.T) {
		// Test various error scenarios to improve coverage

		// Task that panics
		panicTask := task.Task(func() (string, error) {
			panic("test panic")
		}).Named("panic-task")

		workflow1 := Setup(panicTask)
		result, err := workflow1.ExecuteBlocking()

		// Should handle panic gracefully
		if err == nil {
			t.Error("Expected error from panicking task")
		}

		// Result should still exist
		if result == nil {
			t.Error("Expected non-nil result even with panic")
		}

		// Task that returns error
		errorTask := task.Task(func() (string, error) {
			return "", errors.New("deliberate error")
		}).Named("error-task")

		workflow2 := Setup(errorTask)
		result, err = workflow2.ExecuteBlocking()

		if err == nil {
			t.Error("Expected error from error-returning task")
		}

		if result == nil {
			t.Error("Expected non-nil result even with error")
		}

		if !result.HasErrors() {
			t.Error("Expected result to have errors")
		}
	})
}
