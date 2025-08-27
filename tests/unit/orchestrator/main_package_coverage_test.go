package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"

	. "github.com/maniartech/orchestrator"
)

// TestExampleFunctions tests all example functions to achieve 100% coverage
func TestExampleFunctions(t *testing.T) {
	t.Run("ConditionalErrorHandlingBehavior", func(t *testing.T) {
		// Test the actual behavior of conditional error handling, not just execution
		condition := func(ctx Context) (bool, error) {
			return false, fmt.Errorf("user_role not found in context")
		}

		trueTask := Task(func(ctx Context) (string, error) {
			return "true branch", nil
		}).Named("true-task")

		falseTask := Task(func(ctx Context) (string, error) {
			return "false branch", nil
		}).Named("false-task")

		conditional := Conditional(condition, trueTask, falseTask).Named("error-conditional")
		workflow := Setup(conditional)

		result, err := workflow.Await()

		// Validate error handling behavior
		if err == nil {
			t.Error("Expected error from condition evaluation failure")
		}

		if result == nil {
			t.Error("Result should not be nil even with error")
		}

		if !result.HasErrors() {
			t.Error("Result should contain errors from condition failure")
		}

		// Validate error message contains condition failure info
		if !strings.Contains(err.Error(), "condition evaluation failed") {
			t.Errorf("Error should mention condition evaluation failure, got: %v", err)
		}
	})

	t.Run("ConditionalSuccessfulExecutionBehavior", func(t *testing.T) {
		// Test successful conditional execution behavior
		condition := func(ctx Context) (bool, error) {
			return true, nil // Morning condition
		}

		trueTask := Task(func(ctx Context) (string, error) {
			return "Good morning! Starting the day.", nil
		}).Named("morning-task")

		falseTask := Task(func(ctx Context) (string, error) {
			return "Good evening! Winding down.", nil
		}).Named("evening-task")

		conditional := Conditional(condition, trueTask, falseTask).Named("time-conditional")
		workflow := Setup(conditional)

		result, err := workflow.Await()

		// Validate successful execution
		if err != nil {
			t.Errorf("Conditional execution should succeed, got error: %v", err)
		}

		if result == nil {
			t.Error("Result should not be nil for successful execution")
		}

		// Validate correct branch execution
		morningResult := result.Get("morning-task")
		if morningResult != "Good morning! Starting the day." {
			t.Errorf("Expected morning message, got: %v", morningResult)
		}

		// Validate false branch was not executed
		eveningResult := result.Get("evening-task")
		if eveningResult != nil {
			t.Errorf("Evening task should not have executed, got: %v", eveningResult)
		}
	})

	t.Run("SimpleTaskLogic", func(t *testing.T) {
		// Test the same logic as Example_simpleTask to improve coverage
		result, err := Setup(
			Task(func(ctx Context) (string, error) {
				return "Hello from orchestrator!", nil
			}).Named("greeting-task"),
		).Await()

		if err != nil {
			t.Errorf("Simple task failed: %v", err)
		}

		if result == nil {
			t.Error("Result should not be nil")
		}

		greeting := result.Get("greeting-task")
		if greeting != "Hello from orchestrator!" {
			t.Errorf("Expected 'Hello from orchestrator!', got %v", greeting)
		}
	})

	t.Run("TaskWithConfigurationLogic", func(t *testing.T) {
		// Test the same logic as Example_taskWithConfiguration
		result, err := Setup(
			Task(func(ctx Context) (int, error) {
				return 42, nil
			}).Named("answer-task"),
		).With(DefaultConfig()).Await()

		if err != nil {
			t.Errorf("Configured task failed: %v", err)
		}

		if result == nil {
			t.Error("Result should not be nil")
		}

		answer := result.Get("answer-task")
		if answer != 42 {
			t.Errorf("Expected 42, got %v", answer)
		}
	})

	t.Run("StatusMonitoringLogic", func(t *testing.T) {
		// Test the same logic as Example_statusMonitoring
		workflow := Setup(
			Task(func(ctx Context) (string, error) {
				return "Task completed", nil
			}).Named("monitored-task"),
		)

		// Test status progression
		initialStatus := workflow.GetStatus()
		if initialStatus != NotStarted {
			t.Errorf("Expected NotStarted, got %v", initialStatus)
		}

		result, err := workflow.Await()
		if err != nil {
			t.Errorf("Monitored task failed: %v", err)
		}

		finalStatus := workflow.GetStatus()
		if !finalStatus.IsTerminal() {
			t.Errorf("Expected terminal status, got %v", finalStatus)
		}

		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("AdvancedConfigurationScenarios", func(t *testing.T) {
		// Test advanced configuration scenarios
		cfg := Config{
			Timeout: 5 * time.Second,
		}

		result, err := Setup(
			Task(func(ctx Context) (string, error) {
				time.Sleep(10 * time.Millisecond)
				return "configured result", nil
			}).Named("advanced-config-task"),
		).With(cfg).Await()

		if err != nil {
			t.Errorf("Advanced configuration task failed: %v", err)
		}

		if result == nil {
			t.Error("Result should not be nil")
		}

		value := result.Get("advanced-config-task")
		if value != "configured result" {
			t.Errorf("Expected 'configured result', got %v", value)
		}
	})

	t.Run("WorkflowEdgeCases", func(t *testing.T) {
		// Test workflow with nil task name
		result, err := Setup(
			Task(func(ctx Context) (string, error) {
				return "unnamed task", nil
			}), // No .Named() call
		).Await()

		if err != nil {
			t.Errorf("Unnamed task failed: %v", err)
		}

		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("ErrorPathVariations", func(t *testing.T) {
		// Test various error paths
		result, err := Setup(
			Task(func(ctx Context) (string, error) {
				return "", fmt.Errorf("test error")
			}).Named("error-task"),
		).Await()

		if err == nil {
			t.Error("Expected error from task")
		}

		if result == nil {
			t.Error("Result should not be nil even with error")
		}

		if !result.HasErrors() {
			t.Error("Result should have errors")
		}
	})
}

// TestUncoveredOrchestratorPaths tests uncovered paths in orchestrator.go
func TestUncoveredOrchestratorPaths(t *testing.T) {
	t.Run("Await_AlreadyStarted", func(t *testing.T) {
		// Test the path where execution is already started
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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

		// Validate manual mode behavior - should have low progress initially
		if progress.Current > progress.Total {
			t.Errorf("Progress current (%d) should not exceed total (%d)", progress.Current, progress.Total)
		}

		// In manual mode, progress should be deterministic
		if progress.Total <= 0 {
			t.Errorf("Progress total should be positive, got: %d", progress.Total)
		}

		// Validate timestamp is recent (within last second)
		if time.Since(progress.Timestamp) > time.Second {
			t.Errorf("Progress timestamp should be recent, got: %v", progress.Timestamp)
		}

		// Wait for completion
		_, _ = workflow.Await()
	})

	t.Run("GetProgress_HybridMode", func(t *testing.T) {
		// Test hybrid progress mode path
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
		// Use a channel to control task execution timing
		startChan := make(chan struct{})
		taskFn := func(ctx Context) (string, error) {
			<-startChan // Wait for signal to proceed
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

		// Wait for workflow to start (deterministic approach)
		maxWait := 100 * time.Millisecond
		checkInterval := 5 * time.Millisecond
		started := false

		for elapsed := time.Duration(0); elapsed < maxWait; elapsed += checkInterval {
			if workflow.IsRunning() {
				started = true
				break
			}
			time.Sleep(checkInterval)
		}

		if !started {
			t.Error("Workflow should be running after Execute()")
		}

		// Allow task to complete
		close(startChan)

		// Wait for completion
		_, _ = workflow.Await()

		// Should not be running after completion
		if workflow.IsRunning() {
			t.Error("Workflow should not be running after completion")
		}
	})

	t.Run("IsCompleted", func(t *testing.T) {
		taskFn := func(ctx Context) (string, error) {
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
		taskFn := func(ctx Context) (string, error) {
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
func TestDefaultConfig_Coverage(t *testing.T) {
	cfg := DefaultConfig()

	// Verify default config has expected values
	// FailFast is 0, which is a valid default value
	if cfg.ErrorStrategy != FailFast {
		t.Errorf("Expected FailFast error strategy, got %v", cfg.ErrorStrategy)
	}
	if cfg.MaxConcurrency == 0 {
		t.Error("Default config should have non-zero max concurrency")
	}
}
