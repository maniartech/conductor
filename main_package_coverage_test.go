package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/task"
)

// TestExampleFunctions tests all example functions to achieve 100% coverage
func TestExampleFunctions(t *testing.T) {
	t.Run("ExampleConditional_ErrorHandling", func(t *testing.T) {
		// This will execute the example function and cover its code
		ExampleConditional_ErrorHandling()
	})

	t.Run("ExampleConditional_SuccessfulExecution", func(t *testing.T) {
		// This will execute the example function and cover its code
		ExampleConditional_SuccessfulExecution()
	})
}

// TestUncoveredOrchestratorPaths tests uncovered paths in orchestrator.go
func TestUncoveredOrchestratorPaths(t *testing.T) {
	t.Run("Await_AlreadyStarted", func(t *testing.T) {
		// Test the path where execution is already started
		taskFn := func() (string, error) {
			time.Sleep(10 * time.Millisecond) // Small delay
			return "result", nil
		}

		workflow := Setup(task.Task(taskFn).Named("already-started-task"))

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Now call Await - this should hit the "already started" path
		result, err := workflow.Await()
		if err != nil {
			t.Errorf("Await failed: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("AwaitWithContext_AlreadyStarted", func(t *testing.T) {
		// Test the path where execution is already started with context
		taskFn := func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "result", nil
		}

		workflow := Setup(task.Task(taskFn).Named("context-started-task"))

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Now call AwaitWithContext - this should hit the "already started" path
		ctx := context.Background()
		result, err := workflow.AwaitWithContext(ctx)
		if err != nil {
			t.Errorf("AwaitWithContext failed: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("Execute_AlreadyStarted", func(t *testing.T) {
		// Test double execution error
		taskFn := func() (string, error) {
			return "result", nil
		}

		workflow := Setup(task.Task(taskFn).Named("double-execute-task"))

		// First execution should succeed
		err1 := workflow.Execute()
		if err1 != nil {
			t.Fatalf("First execution failed: %v", err1)
		}

		// Second execution should fail
		err2 := workflow.Execute()
		if err2 == nil {
			t.Error("Second execution should have failed")
		}
		if err2.Error() != "workflow already started" {
			t.Errorf("Expected 'workflow already started' error, got: %v", err2)
		}
	})

	t.Run("ExecuteBlocking_Error", func(t *testing.T) {
		// Test ExecuteBlocking with execution error
		taskFn := func() (string, error) {
			return "result", nil
		}

		workflow := Setup(task.Task(taskFn).Named("blocking-error-task"))

		// Start execution first to cause error
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("First execution failed: %v", err)
		}

		// ExecuteBlocking should fail because already started
		result, err := workflow.ExecuteBlocking()
		if err == nil {
			t.Error("ExecuteBlocking should have failed")
		}
		if result != nil {
			t.Error("Result should be nil on error")
		}
	})

	t.Run("AwaitWithTimeout_Success", func(t *testing.T) {
		// Test successful timeout case
		taskFn := func() (string, error) {
			return "timeout-result", nil
		}

		workflow := Setup(task.Task(taskFn).Named("timeout-success-task"))

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Await with generous timeout
		result, err := workflow.AwaitWithTimeout(5 * time.Second)
		if err != nil {
			t.Errorf("AwaitWithTimeout failed: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("executeWorkflow_Error", func(t *testing.T) {
		// Test executeWorkflow error path
		taskFn := func() (string, error) {
			return "", fmt.Errorf("task execution error")
		}

		workflow := Setup(task.Task(taskFn).Named("workflow-error-task"))

		result, err := workflow.Await()
		if err == nil {
			t.Error("Expected error from workflow execution")
		}
		if result == nil {
			t.Error("Result should not be nil even on error")
		}
		if !result.HasErrors() {
			t.Error("Result should have errors")
		}
	})

	t.Run("executeOrchestrationTree_Error", func(t *testing.T) {
		// Test orchestration tree execution error path
		taskFn := func() (string, error) {
			return "", fmt.Errorf("orchestration tree error")
		}

		workflow := Setup(task.Task(taskFn).Named("tree-error-task"))

		result, err := workflow.Await()
		if err == nil {
			t.Error("Expected error from orchestration tree execution")
		}
		if result == nil {
			t.Error("Result should not be nil even on error")
		}
	})
}

// TestWorkflowConfigurationPaths tests uncovered configuration paths
func TestWorkflowConfigurationPaths(t *testing.T) {
	t.Run("applyWorkflowConfiguration_AllFields", func(t *testing.T) {
		// Test all configuration inheritance paths
		taskFn := func() (string, error) {
			return "config-result", nil
		}

		cfg := config.Config{
			ErrorStrategy:  1, // Non-zero value
			Timeout:        5 * time.Second,
			MaxConcurrency: 10,
			Context:        context.Background(),
		}

		workflow := Setup(task.Task(taskFn).Named("config-task")).With(cfg)

		result, err := workflow.Await()
		if err != nil {
			t.Errorf("Configuration test failed: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("applyWorkflowConfiguration_Defaults", func(t *testing.T) {
		// Test default configuration application
		taskFn := func() (string, error) {
			return "default-config-result", nil
		}

		// Use empty config to trigger defaults
		workflow := Setup(task.Task(taskFn).Named("default-config-task"))

		result, err := workflow.Await()
		if err != nil {
			t.Errorf("Default configuration test failed: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})
}

// TestProgressTrackingPaths tests uncovered progress tracking paths
func TestProgressTrackingPaths(t *testing.T) {
	t.Run("GetProgress_ManualMode", func(t *testing.T) {
		// Test manual progress mode path
		taskFn := func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "progress-result", nil
		}

		workflow := Setup(task.Task(taskFn).Named("manual-progress-task"))

		// Set to manual mode
		workflow.SetProgressMode(ProgressModeManual)

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Get progress in manual mode (should hit the manual mode path)
		progress := workflow.GetProgress()
		if progress.Message != "Manual progress mode - no progress reported" {
			t.Errorf("Expected manual progress message, got: %s", progress.Message)
		}

		// Wait for completion
		_, _ = workflow.Await()
	})

	t.Run("GetProgress_HybridMode", func(t *testing.T) {
		// Test hybrid progress mode path
		taskFn := func() (string, error) {
			return "hybrid-result", nil
		}

		workflow := Setup(task.Task(taskFn).Named("hybrid-progress-task"))

		// Set to hybrid mode
		workflow.SetProgressMode(ProgressModeHybrid)

		// Report manual progress
		workflow.ReportProgress(50, 100, "Manual progress in hybrid mode")

		// Get progress (should return manual progress)
		progress := workflow.GetProgress()
		if progress.Current != 50 {
			t.Errorf("Expected current progress 50, got: %d", progress.Current)
		}
		if progress.Message != "Manual progress in hybrid mode" {
			t.Errorf("Expected manual progress message, got: %s", progress.Message)
		}
	})

	t.Run("ReportProgress_WithStage", func(t *testing.T) {
		// Test progress reporting with stage
		taskFn := func() (string, error) {
			return "stage-result", nil
		}

		workflow := Setup(task.Task(taskFn).Named("stage-progress-task"))

		// Set stage first
		workflow.SetStage("Processing")

		// Report progress
		workflow.ReportProgress(75, 100, "Progress with stage")

		// Get progress (should include stage)
		progress := workflow.GetProgress()
		if progress.Stage != "Processing" {
			t.Errorf("Expected stage 'Processing', got: %s", progress.Stage)
		}
		if progress.Current != 75 {
			t.Errorf("Expected current progress 75, got: %d", progress.Current)
		}
	})
}

// TestErrorHandlingPaths tests uncovered error handling paths
func TestErrorHandlingPaths(t *testing.T) {
	t.Run("enhanceErrorWithMetadata", func(t *testing.T) {
		// Test error enhancement
		taskFn := func() (string, error) {
			return "", fmt.Errorf("test error for enhancement")
		}

		workflow := Setup(task.Task(taskFn).Named("error-enhancement-task"))

		result, err := workflow.Await()
		if err == nil {
			t.Error("Expected error")
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
		if !result.HasErrors() {
			t.Error("Result should have errors")
		}
	})

	t.Run("wrapExecutionError", func(t *testing.T) {
		// Test error wrapping
		taskFn := func() (string, error) {
			return "", fmt.Errorf("wrapped error test")
		}

		workflow := Setup(task.Task(taskFn).Named("error-wrapping-task"))

		result, err := workflow.Await()
		if err == nil {
			t.Error("Expected wrapped error")
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("wrapExecutionError_UnnamedTask", func(t *testing.T) {
		// Test error wrapping with unnamed task
		taskFn := func() (string, error) {
			return "", fmt.Errorf("unnamed task error")
		}

		// Create task without name
		workflow := Setup(task.Task(taskFn))

		result, err := workflow.Await()
		if err == nil {
			t.Error("Expected wrapped error")
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})
}

// TestResourceTrackerPaths tests resource tracker paths
func TestResourceTrackerPaths(t *testing.T) {
	t.Run("resourceTracker_Operations", func(t *testing.T) {
		// Test resource tracker functionality
		taskFn := func() (string, error) {
			return "resource-result", nil
		}

		workflow := Setup(task.Task(taskFn).Named("resource-task"))

		result, err := workflow.Await()
		if err != nil {
			t.Errorf("Resource tracker test failed: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})
}

// TestStatusMethods tests status-related methods
func TestStatusMethods(t *testing.T) {
	t.Run("Status_String", func(t *testing.T) {
		// Test all status string representations
		statuses := []Status{NotStarted, Running, Completed, Cancelled, Failed}
		expected := []string{"NotStarted", "Running", "Completed", "Cancelled", "Failed"}

		for i, status := range statuses {
			if status.String() != expected[i] {
				t.Errorf("Status %d: expected %s, got %s", i, expected[i], status.String())
			}
		}

		// Test unknown status
		unknownStatus := Status(999)
		if unknownStatus.String() != "Unknown" {
			t.Errorf("Unknown status: expected 'Unknown', got %s", unknownStatus.String())
		}
	})

	t.Run("Status_IsTerminal", func(t *testing.T) {
		// Test terminal status detection
		if !Completed.IsTerminal() {
			t.Error("Completed should be terminal")
		}
		if !Cancelled.IsTerminal() {
			t.Error("Cancelled should be terminal")
		}
		if !Failed.IsTerminal() {
			t.Error("Failed should be terminal")
		}
		if NotStarted.IsTerminal() {
			t.Error("NotStarted should not be terminal")
		}
		if Running.IsTerminal() {
			t.Error("Running should not be terminal")
		}
	})

	t.Run("Status_IsActive", func(t *testing.T) {
		// Test active status detection
		if !Running.IsActive() {
			t.Error("Running should be active")
		}
		if NotStarted.IsActive() {
			t.Error("NotStarted should not be active")
		}
		if Completed.IsActive() {
			t.Error("Completed should not be active")
		}
		if Cancelled.IsActive() {
			t.Error("Cancelled should not be active")
		}
		if Failed.IsActive() {
			t.Error("Failed should not be active")
		}
	})
}

// TestWorkflowStatusMethods tests workflow status methods
func TestWorkflowStatusMethods(t *testing.T) {
	t.Run("IsRunning", func(t *testing.T) {
		taskFn := func() (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "running-test", nil
		}

		workflow := Setup(task.Task(taskFn).Named("running-test-task"))

		// Should not be running initially
		if workflow.IsRunning() {
			t.Error("Workflow should not be running initially")
		}

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start execution: %v", err)
		}

		// Should be running now
		if !workflow.IsRunning() {
			t.Error("Workflow should be running after Execute()")
		}

		// Wait for completion
		_, _ = workflow.Await()

		// Should not be running after completion
		if workflow.IsRunning() {
			t.Error("Workflow should not be running after completion")
		}
	})

	t.Run("IsCompleted", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "completed-test", nil
		}

		workflow := Setup(task.Task(taskFn).Named("completed-test-task"))

		// Should not be completed initially
		if workflow.IsCompleted() {
			t.Error("Workflow should not be completed initially")
		}

		// Execute and wait
		result, err := workflow.Await()
		if err != nil {
			t.Errorf("Workflow execution failed: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil")
		}

		// Should be completed now
		if !workflow.IsCompleted() {
			t.Error("Workflow should be completed after Await()")
		}
	})

	t.Run("IsInTerminalState", func(t *testing.T) {
		taskFn := func() (string, error) {
			return "terminal-test", nil
		}

		workflow := Setup(task.Task(taskFn).Named("terminal-test-task"))

		// Should not be in terminal state initially
		if workflow.IsInTerminalState() {
			t.Error("Workflow should not be in terminal state initially")
		}

		// Execute and wait
		_, _ = workflow.Await()

		// Should be in terminal state now
		if !workflow.IsInTerminalState() {
			t.Error("Workflow should be in terminal state after completion")
		}
	})
}

// TestDefaultConfig tests the DefaultConfig function
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify default config is not nil
	if cfg.ErrorStrategy == 0 {
		t.Error("Default config should have non-zero error strategy")
	}
	if cfg.MaxConcurrency == 0 {
		t.Error("Default config should have non-zero max concurrency")
	}
}
