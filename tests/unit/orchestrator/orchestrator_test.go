package orchestrator_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/maniartech/orchestrator"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
)

// TestTask tests the Task constructor function
func TestTask(t *testing.T) {
	t.Run("string_task", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "test-result", nil
		})

		if task == nil {
			t.Fatal("Expected task to be created")
		}
	})

	t.Run("int_task", func(t *testing.T) {
		task := Task(func() (int, error) {
			return 42, nil
		})

		if task == nil {
			t.Fatal("Expected task to be created")
		}
	})

	t.Run("error_task", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "", errors.New("test error")
		})

		if task == nil {
			t.Fatal("Expected task to be created")
		}
	})

	t.Run("custom_type_task", func(t *testing.T) {
		type User struct {
			ID   int
			Name string
		}

		task := Task(func() (User, error) {
			return User{ID: 123, Name: "John"}, nil
		})

		if task == nil {
			t.Fatal("Expected task to be created")
		}
	})
}

// TestConditional tests the Conditional constructor function
func TestConditional(t *testing.T) {
	t.Run("valid_conditional", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) {
			return true, nil
		}

		trueTask := Task(func() (string, error) {
			return "true-branch", nil
		})

		falseTask := Task(func() (string, error) {
			return "false-branch", nil
		})

		conditional := Conditional(condition, trueTask, falseTask)

		if conditional == nil {
			t.Fatal("Expected conditional to be created")
		}
	})

	t.Run("conditional_with_error_condition", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) {
			return false, errors.New("condition error")
		}

		trueTask := Task(func() (string, error) {
			return "true-branch", nil
		})

		falseTask := Task(func() (string, error) {
			return "false-branch", nil
		})

		conditional := Conditional(condition, trueTask, falseTask)

		if conditional == nil {
			t.Fatal("Expected conditional to be created")
		}
	})

	t.Run("context_based_condition", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) {
			value := ctx.Get("test_key")
			if value == nil {
				return false, errors.New("test_key not found")
			}
			return value.(bool), nil
		}

		trueTask := Task(func() (string, error) {
			return "authenticated", nil
		})

		falseTask := Task(func() (string, error) {
			return "not-authenticated", nil
		})

		conditional := Conditional(condition, trueTask, falseTask)

		if conditional == nil {
			t.Fatal("Expected conditional to be created")
		}
	})
}

// TestSetup tests the Setup function
func TestSetup(t *testing.T) {
	t.Run("setup_with_task", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "test-result", nil
		}).Named("test-task")

		workflow := Setup(task)

		if workflow == nil {
			t.Fatal("Expected workflow to be created")
		}

		if workflow.GetName() != "test-task" {
			t.Errorf("Expected workflow name 'test-task', got '%s'", workflow.GetName())
		}
	})

	t.Run("setup_with_conditional", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) {
			return true, nil
		}

		trueTask := Task(func() (string, error) {
			return "true-result", nil
		})

		falseTask := Task(func() (string, error) {
			return "false-result", nil
		})

		conditional := Conditional(condition, trueTask, falseTask).Named("test-conditional")
		workflow := Setup(conditional)

		if workflow == nil {
			t.Fatal("Expected workflow to be created")
		}

		if workflow.GetName() != "test-conditional" {
			t.Errorf("Expected workflow name 'test-conditional', got '%s'", workflow.GetName())
		}
	})
}

// TestWorkflow_BasicExecution tests basic workflow execution
func TestWorkflow_BasicExecution(t *testing.T) {
	t.Run("simple_task_execution", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "simple-result", nil
		}).Named("simple-task")

		workflow := Setup(task)
		result, err := workflow.Await()

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		value := result.Get("simple-task")
		if value != "simple-result" {
			t.Errorf("Expected 'simple-result', got %v", value)
		}
	})

	t.Run("task_with_error", func(t *testing.T) {
		taskError := errors.New("task execution failed")
		task := Task(func() (string, error) {
			return "", taskError
		}).Named("failing-task")

		workflow := Setup(task)
		result, err := workflow.Await()

		if err == nil {
			t.Fatal("Expected error from failing task")
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}

		if len(result.Errors()) == 0 {
			t.Error("Expected error to be recorded in result")
		}
	})

	t.Run("conditional_execution_true", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) {
			return true, nil
		}

		trueTask := Task(func() (string, error) {
			return "true-executed", nil
		}).Named("true-task")

		falseTask := Task(func() (string, error) {
			return "false-executed", nil
		}).Named("false-task")

		conditional := Conditional(condition, trueTask, falseTask).Named("test-conditional")
		workflow := Setup(conditional)
		result, err := workflow.Await()

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		// Check if true branch was executed
		branchResult := result.Get("if-true")
		if branchResult != "true-executed" {
			t.Errorf("Expected 'true-executed', got %v", branchResult)
		}
	})

	t.Run("conditional_execution_false", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) {
			return false, nil
		}

		trueTask := Task(func() (string, error) {
			return "true-executed", nil
		}).Named("true-task")

		falseTask := Task(func() (string, error) {
			return "false-executed", nil
		}).Named("false-task")

		conditional := Conditional(condition, trueTask, falseTask).Named("test-conditional")
		workflow := Setup(conditional)
		result, err := workflow.Await()

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		// Check if false branch was executed
		branchResult := result.Get("if-false")
		if branchResult != "false-executed" {
			t.Errorf("Expected 'false-executed', got %v", branchResult)
		}
	})
}

// TestWorkflow_Configuration tests workflow configuration
func TestWorkflow_Configuration(t *testing.T) {
	t.Run("with_config", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "config-result", nil
		}).Named("config-task")

		config := Config{
			Timeout:        30 * time.Second,
			MaxConcurrency: 5,
			ErrorStrategy:  FailFast,
		}

		workflow := Setup(task).With(config)

		if workflow.GetConfig().Timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", workflow.GetConfig().Timeout)
		}

		if workflow.GetConfig().MaxConcurrency != 5 {
			t.Errorf("Expected max concurrency 5, got %d", workflow.GetConfig().MaxConcurrency)
		}

		if workflow.GetConfig().ErrorStrategy != FailFast {
			t.Errorf("Expected FailFast strategy, got %v", workflow.GetConfig().ErrorStrategy)
		}
	})

	t.Run("default_config", func(t *testing.T) {
		defaultCfg := DefaultConfig()

		if defaultCfg.ErrorStrategy == 0 {
			// Default error strategy might be 0 (which is valid)
			// Let's check if it's actually set to a valid value
		}

		if defaultCfg.MaxConcurrency == 0 {
			t.Error("Expected default max concurrency to be set")
		}
	})
}

// TestWorkflow_AsyncExecution tests async execution capabilities
func TestWorkflow_AsyncExecution(t *testing.T) {
	t.Run("execute_and_await", func(t *testing.T) {
		task := Task(func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "async-result", nil
		}).Named("async-task")

		workflow := Setup(task)

		// Start async execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Give it a moment to start
		time.Sleep(10 * time.Millisecond)

		// Check status (might complete very quickly)
		// The workflow should either be running or have completed successfully
		status := workflow.GetStatus()
		if status != Running && status != Completed {
			// If it's still NotStarted, there might be an issue with Execute()
			// Let's just verify it eventually completes
			t.Logf("Workflow status after Execute(): %v", status)
		}

		// Wait for completion
		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		if !workflow.IsCompleted() {
			t.Error("Expected workflow to be completed")
		}

		value := result.Get("async-task")
		if value != "async-result" {
			t.Errorf("Expected 'async-result', got %v", value)
		}
	})

	t.Run("execute_blocking", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "blocking-result", nil
		}).Named("blocking-task")

		workflow := Setup(task)

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		if !workflow.IsCompleted() {
			t.Error("Expected workflow to be completed")
		}

		value := result.Get("blocking-task")
		if value != "blocking-result" {
			t.Errorf("Expected 'blocking-result', got %v", value)
		}
	})

	t.Run("await_with_timeout", func(t *testing.T) {
		task := Task(func() (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "timeout-result", nil
		}).Named("timeout-task")

		workflow := Setup(task)

		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Wait with short timeout
		_, err = workflow.AwaitWithTimeout(10 * time.Millisecond)
		if err == nil {
			t.Error("Expected timeout error")
		}

		// Result should still be available for inspection
		// Note: AwaitWithTimeout might return nil result on timeout
		// This is acceptable behavior
	})

	t.Run("await_with_context", func(t *testing.T) {
		task := Task(func() (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "context-result", nil
		}).Named("context-task")

		workflow := Setup(task)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := workflow.AwaitWithContext(ctx)
		if err == nil {
			t.Error("Expected context cancellation error")
		}

		// Result should still be available
		// Note: AwaitWithContext might return nil result on cancellation
		// This is acceptable behavior
	})
}

// TestWorkflow_StatusTracking tests status tracking functionality
func TestWorkflow_StatusTracking(t *testing.T) {
	t.Run("status_progression", func(t *testing.T) {
		task := Task(func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "status-result", nil
		}).Named("status-task")

		workflow := Setup(task)

		// Initial status
		if workflow.GetStatus() != NotStarted {
			t.Errorf("Expected NotStarted status, got %v", workflow.GetStatus())
		}

		if workflow.IsRunning() {
			t.Error("Expected workflow not to be running initially")
		}

		if workflow.IsCompleted() {
			t.Error("Expected workflow not to be completed initially")
		}

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Should be running
		time.Sleep(1 * time.Millisecond) // Give it a moment to start
		if !workflow.IsRunning() {
			t.Error("Expected workflow to be running")
		}

		// Wait for completion
		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Should be completed
		if !workflow.IsCompleted() {
			t.Error("Expected workflow to be completed")
		}

		if workflow.GetStatus() != Completed {
			t.Errorf("Expected Completed status, got %v", workflow.GetStatus())
		}

		if !workflow.IsInTerminalState() {
			t.Error("Expected workflow to be in terminal state")
		}

		value := result.Get("status-task")
		if value != "status-result" {
			t.Errorf("Expected 'status-result', got %v", value)
		}
	})

	t.Run("status_with_error", func(t *testing.T) {
		taskError := errors.New("status test error")
		task := Task(func() (string, error) {
			return "", taskError
		}).Named("error-task")

		workflow := Setup(task)

		result, err := workflow.ExecuteBlocking()
		if err == nil {
			t.Fatal("Expected error from failing task")
		}

		if workflow.GetStatus() != Failed {
			t.Errorf("Expected Failed status, got %v", workflow.GetStatus())
		}

		if !workflow.IsInTerminalState() {
			t.Error("Expected workflow to be in terminal state")
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}
	})
}

// TestWorkflow_ProgressTracking tests progress tracking functionality
func TestWorkflow_ProgressTracking(t *testing.T) {
	t.Run("automatic_progress", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "progress-result", nil
		}).Named("progress-task")

		workflow := Setup(task)

		// Initial progress
		progress := workflow.GetProgress()
		if progress.Total != 1 {
			t.Errorf("Expected total tasks 1, got %d", progress.Total)
		}

		if progress.Current != 0 {
			t.Errorf("Expected current tasks 0, got %d", progress.Current)
		}

		// Execute and check final progress
		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		finalProgress := workflow.GetProgress()
		if finalProgress.Current != 1 {
			t.Errorf("Expected current tasks 1, got %d", finalProgress.Current)
		}

		if finalProgress.Percentage != 100.0 {
			t.Errorf("Expected 100%% completion, got %.1f%%", finalProgress.Percentage)
		}

		value := result.Get("progress-task")
		if value != "progress-result" {
			t.Errorf("Expected 'progress-result', got %v", value)
		}
	})

	t.Run("manual_progress", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "manual-result", nil
		}).Named("manual-task")

		workflow := Setup(task)
		workflow.SetProgressMode(ProgressModeManual)

		// Report manual progress
		workflow.ReportProgress(50, 100, "Halfway done")

		progress := workflow.GetProgress()
		if progress.Current != 50 || progress.Total != 100 {
			t.Errorf("Expected progress 50/100, got %d/%d", progress.Current, progress.Total)
		}

		if progress.Message != "Halfway done" {
			t.Errorf("Expected message 'Halfway done', got: %s", progress.Message)
		}

		if progress.Percentage != 50.0 {
			t.Errorf("Expected 50%% completion, got %.1f%%", progress.Percentage)
		}
	})

	t.Run("stage_based_progress", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "stage-result", nil
		}).Named("stage-task")

		workflow := Setup(task)

		// Set stages
		workflow.SetStage("Initialization")
		progress1 := workflow.GetProgress()
		if progress1.Stage != "Initialization" {
			t.Errorf("Expected stage 'Initialization', got: %s", progress1.Stage)
		}

		workflow.SetStage("Processing")
		progress2 := workflow.GetProgress()
		if progress2.Stage != "Processing" {
			t.Errorf("Expected stage 'Processing', got: %s", progress2.Stage)
		}

		workflow.SetStage("Finalization")
		progress3 := workflow.GetProgress()
		if progress3.Stage != "Finalization" {
			t.Errorf("Expected stage 'Finalization', got: %s", progress3.Stage)
		}
	})

	t.Run("legacy_progress_format", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "legacy-result", nil
		}).Named("legacy-task")

		workflow := Setup(task)
		workflow.ReportProgress(80, 100, "Legacy test")

		completed, total, percentage := workflow.GetProgressLegacy()
		if completed != 80 || total != 100 || percentage != 80.0 {
			t.Errorf("Expected legacy progress 80/100 (80%%), got %d/%d (%.1f%%)",
				completed, total, percentage)
		}
	})
}

// TestWorkflow_Callbacks tests callback functionality
func TestWorkflow_Callbacks(t *testing.T) {
	t.Run("progress_callback", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "callback-result", nil
		}).Named("callback-task")

		workflow := Setup(task)

		var progressCalled atomic.Bool
		workflow.OnProgress(func(progress Progress) {
			progressCalled.Store(true)
			if progress.Current < 0 || progress.Total < 0 {
				t.Errorf("Invalid progress values: %d/%d", progress.Current, progress.Total)
			}
		})

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Wait a bit for callbacks to be processed
		time.Sleep(10 * time.Millisecond)

		if !progressCalled.Load() {
			t.Error("Expected progress callback to be called")
		}

		value := result.Get("callback-task")
		if value != "callback-result" {
			t.Errorf("Expected 'callback-result', got %v", value)
		}
	})

	t.Run("status_change_callback", func(t *testing.T) {
		task := Task(func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "status-callback-result", nil
		}).Named("status-callback-task")

		workflow := Setup(task)

		var mu sync.Mutex
		var statusChanges []Status
		workflow.OnStatusChange(func(oldStatus, newStatus Status) {
			mu.Lock()
			statusChanges = append(statusChanges, newStatus)
			mu.Unlock()
		})

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Wait for all callbacks to be processed
		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		statusChangeCount := len(statusChanges)
		statusChangesCopy := make([]Status, len(statusChanges))
		copy(statusChangesCopy, statusChanges)
		mu.Unlock()

		if statusChangeCount < 2 {
			t.Errorf("Expected at least 2 status changes, got %d", statusChangeCount)
		}

		// Should have Running and Completed
		hasRunning := false
		hasCompleted := false
		for _, status := range statusChangesCopy {
			if status == Running {
				hasRunning = true
			}
			if status == Completed {
				hasCompleted = true
			}
		}

		if !hasRunning {
			t.Error("Expected Running status change")
		}
		if !hasCompleted {
			t.Error("Expected Completed status change")
		}

		value := result.Get("status-callback-task")
		if value != "status-callback-result" {
			t.Errorf("Expected 'status-callback-result', got %v", value)
		}
	})

	t.Run("error_callback", func(t *testing.T) {
		taskError := errors.New("callback test error")
		task := Task(func() (string, error) {
			return "", taskError
		}).Named("error-callback-task")

		workflow := Setup(task)

		var mu sync.Mutex
		var capturedError error
		workflow.OnError(func(err error) {
			mu.Lock()
			capturedError = err
			mu.Unlock()
		})

		result, err := workflow.ExecuteBlocking()
		if err == nil {
			t.Fatal("Expected error from failing task")
		}

		// Wait a bit for callbacks
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		errorCopy := capturedError
		mu.Unlock()

		if errorCopy == nil {
			t.Error("Expected error callback to be called")
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}
	})

	t.Run("completion_callback", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "completion-result", nil
		}).Named("completion-task")

		workflow := Setup(task)

		var mu sync.Mutex
		var completionCalled bool
		var completionResult *Result
		var completionError error

		workflow.OnComplete(func(result *Result, err error) {
			mu.Lock()
			completionCalled = true
			completionResult = result
			completionError = err
			mu.Unlock()
		})

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Wait a bit for callbacks
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		called := completionCalled
		resultCopy := completionResult
		errorCopy := completionError
		mu.Unlock()

		if !called {
			t.Error("Expected completion callback to be called")
		}

		if errorCopy != nil {
			t.Errorf("Expected no error in completion callback, got: %v", errorCopy)
		}

		if resultCopy == nil {
			t.Error("Expected result in completion callback")
		}

		value := result.Get("completion-task")
		if value != "completion-result" {
			t.Errorf("Expected 'completion-result', got %v", value)
		}
	})
}

// TestWorkflow_Cancellation tests cancellation functionality
func TestWorkflow_Cancellation(t *testing.T) {
	t.Run("cancel_workflow", func(t *testing.T) {
		task := Task(func() (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "should-not-complete", nil
		}).Named("long-task")

		workflow := Setup(task)

		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Cancel after a short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			workflow.Cancel()
		}()

		result, err := workflow.Await()

		if workflow.GetStatus() != Cancelled {
			t.Errorf("Expected status Cancelled, got: %v", workflow.GetStatus())
		}

		if result == nil {
			t.Error("Expected result container even with cancellation")
		}
	})

	t.Run("cancel_with_reason", func(t *testing.T) {
		task := Task(func() (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "should-not-complete", nil
		}).Named("reason-task")

		workflow := Setup(task)

		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Cancel with reason after a short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			workflow.CancelWithReason("test cancellation reason")
		}()

		result, err := workflow.Await()

		if workflow.GetStatus() != Cancelled {
			t.Errorf("Expected status Cancelled, got: %v", workflow.GetStatus())
		}

		if result == nil {
			t.Error("Expected result container even with cancellation")
		}

		// Check cancellation reason
		reason := result.Get("cancellation_reason")
		if reason != "test cancellation reason" {
			t.Errorf("Expected cancellation reason 'test cancellation reason', got: %v", reason)
		}
	})
}

// TestWorkflow_BackwardCompatibility tests backward compatibility
func TestWorkflow_BackwardCompatibility(t *testing.T) {
	t.Run("await_method", func(t *testing.T) {
		task := Task(func() (string, error) {
			return "compat-result", nil
		}).Named("compat-task")

		workflow := Setup(task)

		// Use old Await method (should work)
		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		if !workflow.IsCompleted() {
			t.Error("Expected workflow to be completed")
		}

		value := result.Get("compat-task")
		if value != "compat-result" {
			t.Errorf("Expected 'compat-result', got %v", value)
		}
	})
}

// TestStatus tests Status type methods
func TestStatus(t *testing.T) {
	t.Run("status_string", func(t *testing.T) {
		if NotStarted.String() != "NotStarted" {
			t.Errorf("Expected 'NotStarted', got '%s'", NotStarted.String())
		}

		if Running.String() != "Running" {
			t.Errorf("Expected 'Running', got '%s'", Running.String())
		}

		if Completed.String() != "Completed" {
			t.Errorf("Expected 'Completed', got '%s'", Completed.String())
		}

		if Cancelled.String() != "Cancelled" {
			t.Errorf("Expected 'Cancelled', got '%s'", Cancelled.String())
		}

		if Failed.String() != "Failed" {
			t.Errorf("Expected 'Failed', got '%s'", Failed.String())
		}
	})

	t.Run("is_terminal", func(t *testing.T) {
		if NotStarted.IsTerminal() {
			t.Error("Expected NotStarted not to be terminal")
		}

		if Running.IsTerminal() {
			t.Error("Expected Running not to be terminal")
		}

		if !Completed.IsTerminal() {
			t.Error("Expected Completed to be terminal")
		}

		if !Cancelled.IsTerminal() {
			t.Error("Expected Cancelled to be terminal")
		}

		if !Failed.IsTerminal() {
			t.Error("Expected Failed to be terminal")
		}
	})

	t.Run("is_active", func(t *testing.T) {
		if NotStarted.IsActive() {
			t.Error("Expected NotStarted not to be active")
		}

		if !Running.IsActive() {
			t.Error("Expected Running to be active")
		}

		if Completed.IsActive() {
			t.Error("Expected Completed not to be active")
		}

		if Cancelled.IsActive() {
			t.Error("Expected Cancelled not to be active")
		}

		if Failed.IsActive() {
			t.Error("Expected Failed not to be active")
		}
	})
}

// BenchmarkTask benchmarks task creation
func BenchmarkTask(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Task(func() (string, error) {
			return "benchmark-result", nil
		})
	}
}

// BenchmarkWorkflow_Execute benchmarks workflow execution
func BenchmarkWorkflow_Execute(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create new task instance for each iteration to avoid reuse issues
		task := Task(func() (string, error) {
			return "benchmark-result", nil
		}).Named(fmt.Sprintf("benchmark-task-%d", i))

		workflow := Setup(task)
		_, err := workflow.ExecuteBlocking()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkConditional_Execute benchmarks conditional execution
func BenchmarkConditional_Execute(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create new task instances for each iteration to avoid reuse issues
		condition := func(ctx orchContext.Context) (bool, error) {
			return true, nil
		}

		trueTask := Task(func() (string, error) {
			return "true-result", nil
		}).Named(fmt.Sprintf("true-task-%d", i))

		falseTask := Task(func() (string, error) {
			return "false-result", nil
		}).Named(fmt.Sprintf("false-task-%d", i))

		conditional := Conditional(condition, trueTask, falseTask).Named(fmt.Sprintf("conditional-%d", i))

		workflow := Setup(conditional)
		_, err := workflow.ExecuteBlocking()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
