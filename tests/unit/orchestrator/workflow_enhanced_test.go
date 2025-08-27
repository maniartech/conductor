package orchestrator_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/orchestrator"
	. "github.com/maniartech/orchestrator"
)

// TestEnhancedWorkflowAPI tests the new async execution and progress tracking features
func TestEnhancedWorkflowAPI(t *testing.T) {
	t.Run("async execution", func(t *testing.T) {
		// Create a simple task with longer delay
		task := Task(func(ctx orchestrator.Context) (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "async-result", nil
		}).Named("async-task")

		// Create workflow
		workflow := Setup(task)

		// Start async execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Give it a moment to start
		time.Sleep(5 * time.Millisecond)

		// Check status
		if !workflow.IsRunning() {
			t.Errorf("Expected workflow to be running, got status: %v", workflow.GetStatus())
		}

		// Wait for completion
		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Check final status
		if !workflow.IsCompleted() {
			t.Error("Expected workflow to be completed")
		}

		// Verify result
		value := result.Get("async-task")
		if value != "async-result" {
			t.Errorf("Expected 'async-result', got: %v", value)
		}
	})

	t.Run("progress tracking", func(t *testing.T) {
		// Create a task
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "progress-result", nil
		}).Named("progress-task")

		// Create workflow
		workflow := Setup(task)

		// Set up progress callback
		var progressCalled atomic.Bool
		workflow.OnProgress(func(progress Progress) {
			progressCalled.Store(true)
			if progress.Current < 0 || progress.Total < 0 {
				t.Errorf("Invalid progress values: %d/%d", progress.Current, progress.Total)
			}
		})

		// Execute workflow
		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Wait a bit for callbacks to be processed
		time.Sleep(10 * time.Millisecond)

		// Verify progress callback was called
		if !progressCalled.Load() {
			t.Error("Expected progress callback to be called")
		}

		// Verify result
		value := result.Get("progress-task")
		if value != "progress-result" {
			t.Errorf("Expected 'progress-result', got: %v", value)
		}
	})

	t.Run("manual progress reporting", func(t *testing.T) {
		// Create a task
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "manual-result", nil
		}).Named("manual-task")

		// Create workflow
		workflow := Setup(task)

		// Set manual progress mode
		workflow.SetProgressMode(ProgressModeManual)

		// Set up progress callback
		var mu sync.Mutex
		var lastProgress Progress
		workflow.OnProgress(func(progress Progress) {
			mu.Lock()
			lastProgress = progress
			mu.Unlock()
		})

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Report manual progress
		workflow.ReportProgress(50, 100, "Halfway done")

		// Wait a bit for callback
		time.Sleep(10 * time.Millisecond)

		// Check progress
		mu.Lock()
		current := lastProgress.Current
		total := lastProgress.Total
		message := lastProgress.Message
		mu.Unlock()

		if current != 50 || total != 100 {
			t.Errorf("Expected progress 50/100, got %d/%d", current, total)
		}

		if message != "Halfway done" {
			t.Errorf("Expected message 'Halfway done', got: %s", message)
		}

		// Wait for completion
		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Verify result
		value := result.Get("manual-task")
		if value != "manual-result" {
			t.Errorf("Expected 'manual-result', got: %v", value)
		}
	})

	t.Run("stage-based progress", func(t *testing.T) {
		// Create a task
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "stage-result", nil
		}).Named("stage-task")

		// Create workflow
		workflow := Setup(task)

		// Set up progress callback
		var mu sync.Mutex
		var stages []string
		workflow.OnProgress(func(progress Progress) {
			if progress.Stage != "" {
				mu.Lock()
				stages = append(stages, progress.Stage)
				mu.Unlock()
			}
		})

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Set stages
		workflow.SetStage("Initialization")
		workflow.SetStage("Processing")
		workflow.SetStage("Finalization")

		// Wait for completion
		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Verify stages were captured
		mu.Lock()
		stageCount := len(stages)
		mu.Unlock()

		if stageCount == 0 {
			t.Error("Expected stage updates to be captured")
		}

		// Verify result
		value := result.Get("stage-task")
		if value != "stage-result" {
			t.Errorf("Expected 'stage-result', got: %v", value)
		}
	})

	t.Run("status change callbacks", func(t *testing.T) {
		// Create a task
		task := Task(func(ctx orchestrator.Context) (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "status-result", nil
		}).Named("status-task")

		// Create workflow
		workflow := Setup(task)

		// Track status changes
		var mu sync.Mutex
		var statusChanges []Status
		workflow.OnStatusChange(func(oldStatus, newStatus Status) {
			mu.Lock()
			statusChanges = append(statusChanges, newStatus)
			mu.Unlock()
		})

		// Execute workflow
		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Wait for all callbacks to be processed
		time.Sleep(20 * time.Millisecond)

		// Verify status changes
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

		// Verify result
		value := result.Get("status-task")
		if value != "status-result" {
			t.Errorf("Expected 'status-result', got: %v", value)
		}
	})

	t.Run("error handling with callbacks", func(t *testing.T) {
		// Create a failing task
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "", errors.New("test error")
		}).Named("error-task")

		// Create workflow
		workflow := Setup(task)

		// Track errors
		var mu sync.Mutex
		var capturedError error
		workflow.OnError(func(err error) {
			mu.Lock()
			capturedError = err
			mu.Unlock()
		})

		// Track completion
		var completionCalled bool
		var completionError error
		workflow.OnComplete(func(result *Result, err error) {
			mu.Lock()
			completionCalled = true
			completionError = err
			mu.Unlock()
		})

		// Execute workflow
		result, err := workflow.ExecuteBlocking()

		// Should have error
		if err == nil {
			t.Error("Expected workflow to fail")
		}

		// Wait a bit for callbacks
		time.Sleep(10 * time.Millisecond)

		// Verify error callback was called
		mu.Lock()
		errorCopy := capturedError
		called := completionCalled
		_ = completionError // Ignore completion error for this test
		mu.Unlock()

		if errorCopy == nil {
			t.Error("Expected error callback to be called")
		}

		// Verify completion callback was called
		if !called {
			t.Error("Expected completion callback to be called")
		}

		if completionError == nil {
			t.Error("Expected completion callback to receive error")
		}

		// Verify result container exists even with error
		if result == nil {
			t.Error("Expected result container even with error")
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		// Create a long-running task
		task := Task(func(ctx orchestrator.Context) (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "should-not-complete", nil
		}).Named("long-task")

		// Create workflow
		workflow := Setup(task)

		// Start execution
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Cancel after a short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			workflow.CancelWithReason("test cancellation")
		}()

		// Wait for completion
		result, err := workflow.Await()

		// Should be cancelled
		if workflow.GetStatus() != Cancelled {
			t.Errorf("Expected status Cancelled, got: %v", workflow.GetStatus())
		}

		// Result should still be available
		if result == nil {
			t.Error("Expected result container even with cancellation")
		}

		// Check cancellation reason
		reason := result.Get("cancellation_reason")
		if reason != "test cancellation" {
			t.Errorf("Expected cancellation reason 'test cancellation', got: %v", reason)
		}
	})

	t.Run("backward compatibility", func(t *testing.T) {
		// Create a task
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "compat-result", nil
		}).Named("compat-task")

		// Create workflow
		workflow := Setup(task)

		// Use old Await method (should work)
		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Verify result
		value := result.Get("compat-task")
		if value != "compat-result" {
			t.Errorf("Expected 'compat-result', got: %v", value)
		}

		// Should be completed
		if !workflow.IsCompleted() {
			t.Error("Expected workflow to be completed")
		}
	})
}

// TestHybridProgressTracking tests the hybrid progress tracking approach
func TestHybridProgressTracking(t *testing.T) {
	t.Run("automatic mode", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "auto-result", nil
		}).Named("auto-task")

		workflow := Setup(task)

		// Should default to automatic mode
		progress := workflow.GetProgress()
		if progress.Total != 1 {
			t.Errorf("Expected total tasks 1, got: %d", progress.Total)
		}
	})

	t.Run("manual mode", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "manual-result", nil
		}).Named("manual-task")

		workflow := Setup(task)
		workflow.SetProgressMode(ProgressModeManual)

		// Report manual progress
		workflow.ReportProgress(75, 100, "Almost done")

		progress := workflow.GetProgress()
		if progress.Current != 75 || progress.Total != 100 {
			t.Errorf("Expected progress 75/100, got %d/%d", progress.Current, progress.Total)
		}

		if progress.Message != "Almost done" {
			t.Errorf("Expected message 'Almost done', got: %s", progress.Message)
		}
	})

	t.Run("hybrid mode", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "hybrid-result", nil
		}).Named("hybrid-task")

		workflow := Setup(task)
		workflow.SetProgressMode(ProgressModeHybrid)

		// Initially should use automatic progress
		progress := workflow.GetProgress()
		if progress.Total != 1 {
			t.Errorf("Expected automatic total 1, got: %d", progress.Total)
		}

		// Report manual progress (should override)
		workflow.ReportProgress(60, 100, "Manual override")

		progress = workflow.GetProgress()
		if progress.Current != 60 || progress.Total != 100 {
			t.Errorf("Expected manual progress 60/100, got %d/%d", progress.Current, progress.Total)
		}

		if progress.Message != "Manual override" {
			t.Errorf("Expected message 'Manual override', got: %s", progress.Message)
		}
	})

	t.Run("legacy progress format", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "legacy-result", nil
		}).Named("legacy-task")

		workflow := Setup(task)
		workflow.ReportProgress(80, 100, "Legacy test")

		// Test legacy format
		completed, total, percentage := workflow.GetProgressLegacy()
		if completed != 80 || total != 100 || percentage != 80.0 {
			t.Errorf("Expected legacy progress 80/100 (80%%), got %d/%d (%.1f%%)",
				completed, total, percentage)
		}
	})
}
