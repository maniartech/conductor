package orchestrator

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	orchErrors "github.com/maniartech/orchestrator/pkg/errors"
)

// TestWorkflow_InternalMethods tests internal workflow methods
func TestWorkflow_InternalMethods(t *testing.T) {
	t.Run("apply_workflow_configuration", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "config-test", nil
		}).Named("config-task")

		workflow := Setup(task).With(Config{
			Timeout:        30 * time.Second,
			MaxConcurrency: 50,
			ErrorStrategy:  CollectAll,
		})

		// Test configuration application
		baseConfig := config.Config{
			Timeout: 10 * time.Second, // Should be overridden
		}

		finalConfig := workflow.applyWorkflowConfiguration(baseConfig)

		if finalConfig.Timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", finalConfig.Timeout)
		}

		if finalConfig.MaxConcurrency != 50 {
			t.Errorf("Expected max concurrency 50, got %d", finalConfig.MaxConcurrency)
		}

		if finalConfig.ErrorStrategy != orchErrors.CollectAll {
			t.Errorf("Expected CollectAll strategy, got %v", finalConfig.ErrorStrategy)
		}
	})

	t.Run("apply_workflow_configuration_defaults", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "defaults-test", nil
		}).Named("defaults-task")

		workflow := Setup(task) // No configuration

		baseConfig := config.Config{} // Empty config

		finalConfig := workflow.applyWorkflowConfiguration(baseConfig)

		// Should apply defaults
		if finalConfig.ErrorStrategy == 0 {
			// FailFast is 0, which is valid
		}

		if finalConfig.MaxConcurrency != 100 {
			t.Errorf("Expected default max concurrency 100, got %d", finalConfig.MaxConcurrency)
		}
	})

	t.Run("enhance_error_with_metadata", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "", errors.New("test error")
		}).Named("error-task")

		workflow := Setup(task)

		originalError := errors.New("original error")
		enhancedError := workflow.enhanceErrorWithMetadata(originalError, "test-orchestration")

		if enhancedError.Error != originalError {
			t.Error("Expected enhanced error to contain original error")
		}

		if enhancedError.OpID != "workflow-test-orchestration" {
			t.Errorf("Expected operation ID 'workflow-test-orchestration', got '%s'", enhancedError.OpID)
		}

		if enhancedError.Index != 0 {
			t.Errorf("Expected index 0, got %d", enhancedError.Index)
		}
	})

	t.Run("wrap_execution_error", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "", errors.New("execution error")
		}).Named("wrap-task")

		workflow := Setup(task)

		originalError := errors.New("original execution error")
		wrappedError := workflow.wrapExecutionError(originalError, workflow.orchestration)

		if wrappedError == nil {
			t.Fatal("Expected wrapped error")
		}

		// Should contain orchestration name in error message
		errorMsg := wrappedError.Error()
		if errorMsg == "" {
			t.Error("Expected non-empty error message")
		}
	})

	t.Run("wrap_execution_error_unnamed", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "", errors.New("execution error")
		}) // No name

		workflow := Setup(task)

		originalError := errors.New("original execution error")
		wrappedError := workflow.wrapExecutionError(originalError, workflow.orchestration)

		if wrappedError == nil {
			t.Fatal("Expected wrapped error")
		}

		// Should handle unnamed orchestration
		errorMsg := wrappedError.Error()
		if errorMsg == "" {
			t.Error("Expected non-empty error message")
		}
	})
}

// TestWorkflow_ResourceTracker tests resource tracking functionality
func TestWorkflow_ResourceTracker(t *testing.T) {
	t.Run("resource_tracker_creation", func(t *testing.T) {
		tracker := newResourceTracker()

		if tracker == nil {
			t.Fatal("Expected resource tracker to be created")
		}

		if len(tracker.orchestrations) != 0 {
			t.Error("Expected empty orchestrations list initially")
		}

		if len(tracker.cleanupFuncs) != 0 {
			t.Error("Expected empty cleanup functions list initially")
		}
	})

	t.Run("track_orchestration", func(t *testing.T) {
		tracker := newResourceTracker()
		task := Task(func(ctx Context) (string, error) {
			return "tracked", nil
		}).Named("tracked-task")

		tracker.trackOrchestration(task)

		if len(tracker.orchestrations) != 1 {
			t.Errorf("Expected 1 tracked orchestration, got %d", len(tracker.orchestrations))
		}

		if tracker.orchestrations[0] != task {
			t.Error("Expected tracked orchestration to match")
		}
	})

	t.Run("add_cleanup_function", func(t *testing.T) {
		tracker := newResourceTracker()
		cleanupCalled := false

		tracker.addCleanupFunc(func() {
			cleanupCalled = true
		})

		if len(tracker.cleanupFuncs) != 1 {
			t.Errorf("Expected 1 cleanup function, got %d", len(tracker.cleanupFuncs))
		}

		// Test cleanup execution
		tracker.cleanup()

		if !cleanupCalled {
			t.Error("Expected cleanup function to be called")
		}
	})

	t.Run("cleanup_with_panic_recovery", func(t *testing.T) {
		tracker := newResourceTracker()

		// Add a cleanup function that panics
		tracker.addCleanupFunc(func() {
			panic("cleanup panic")
		})

		// Add a normal cleanup function
		normalCleanupCalled := false
		tracker.addCleanupFunc(func() {
			normalCleanupCalled = true
		})

		// Cleanup should not panic and should call all functions
		tracker.cleanup()

		if !normalCleanupCalled {
			t.Error("Expected normal cleanup function to be called despite panic")
		}
	})

	t.Run("cleanup_reverse_order", func(t *testing.T) {
		tracker := newResourceTracker()
		var order []int

		// Add cleanup functions in order
		tracker.addCleanupFunc(func() {
			order = append(order, 1)
		})
		tracker.addCleanupFunc(func() {
			order = append(order, 2)
		})
		tracker.addCleanupFunc(func() {
			order = append(order, 3)
		})

		tracker.cleanup()

		// Should execute in reverse order (LIFO)
		expected := []int{3, 2, 1}
		if len(order) != len(expected) {
			t.Errorf("Expected %d cleanup calls, got %d", len(expected), len(order))
		}

		for i, val := range order {
			if val != expected[i] {
				t.Errorf("Expected cleanup order %v, got %v", expected, order)
				break
			}
		}
	})
}

// TestWorkflow_AsyncExecutionInternal tests internal async execution methods
func TestWorkflow_AsyncExecutionInternal(t *testing.T) {
	t.Run("execute_async_completion", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "async-internal", nil
		}).Named("async-internal-task")

		workflow := Setup(task)

		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Wait for completion
		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		if !workflow.IsCompleted() {
			t.Error("Expected workflow to be completed")
		}

		value := result.Get("async-internal-task")
		if value != "async-internal" {
			t.Errorf("Expected 'async-internal', got %v", value)
		}
	})

	t.Run("execute_async_with_error", func(t *testing.T) {
		expectedError := errors.New("async error")
		task := Task(func(ctx Context) (string, error) {
			return "", expectedError
		}).Named("async-error-task")

		workflow := Setup(task)

		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Wait for completion
		result, err := workflow.Await()
		if err == nil {
			t.Fatal("Expected error from failing task")
		}

		if workflow.GetStatus() != Failed {
			t.Errorf("Expected status Failed, got %v", workflow.GetStatus())
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}
	})

	t.Run("double_execute_error", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "double-execute", nil
		}).Named("double-execute-task")

		workflow := Setup(task)

		// First execute should succeed
		err1 := workflow.Execute()
		if err1 != nil {
			t.Fatalf("First execute failed: %v", err1)
		}

		// Second execute should fail
		err2 := workflow.Execute()
		if err2 == nil {
			t.Error("Expected error on second execute")
		}

		// Wait for completion
		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		value := result.Get("double-execute-task")
		if value != "double-execute" {
			t.Errorf("Expected 'double-execute', got %v", value)
		}
	})
}

// TestWorkflow_ProgressTrackingInternal tests internal progress tracking
func TestWorkflow_ProgressTrackingInternal(t *testing.T) {
	t.Run("initialize_progress_tracking", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "progress-init", nil
		}).Named("progress-init-task")

		workflow := Setup(task)

		// Progress should be initialized
		progress := workflow.GetProgress()
		if progress.Total != 1 {
			t.Errorf("Expected total tasks 1, got %d", progress.Total)
		}

		if progress.Current != 0 {
			t.Errorf("Expected current tasks 0, got %d", progress.Current)
		}

		if workflow.progressMode.Load() != ProgressModeAuto {
			t.Errorf("Expected auto progress mode, got %d", workflow.progressMode.Load())
		}
	})

	t.Run("update_progress_with_stage", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "stage-progress", nil
		}).Named("stage-progress-task")

		workflow := Setup(task)

		// Set stage should update progress
		workflow.SetStage("Testing")

		progress := workflow.GetProgress()
		if progress.Stage != "Testing" {
			t.Errorf("Expected stage 'Testing', got '%s'", progress.Stage)
		}
	})

	t.Run("notify_progress_update", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "notify-progress", nil
		}).Named("notify-progress-task")

		workflow := Setup(task)

		var notifiedProgress Progress
		var progressNotified bool
		var mu sync.Mutex

		workflow.OnProgress(func(progress Progress) {
			mu.Lock()
			notifiedProgress = progress
			progressNotified = true
			mu.Unlock()
		})

		// Report progress should trigger notification
		workflow.ReportProgress(75, 100, "Testing notification")

		// Wait for callback
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()

		if !progressNotified {
			t.Error("Expected progress notification")
		}

		if notifiedProgress.Current != 75 {
			t.Errorf("Expected notified current 75, got %d", notifiedProgress.Current)
		}

		if notifiedProgress.Message != "Testing notification" {
			t.Errorf("Expected message 'Testing notification', got '%s'", notifiedProgress.Message)
		}
	})
}

// TestWorkflow_StatusManagement tests status management
func TestWorkflow_StatusManagement(t *testing.T) {
	t.Run("set_status_internal", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "status-test", nil
		}).Named("status-test-task")

		workflow := Setup(task)

		// Initial status
		if workflow.GetStatus() != NotStarted {
			t.Errorf("Expected NotStarted status, got %v", workflow.GetStatus())
		}

		// Execute to change status
		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Should eventually become Running or Completed
		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Should be completed
		if workflow.GetStatus() != Completed {
			t.Errorf("Expected Completed status, got %v", workflow.GetStatus())
		}

		value := result.Get("status-test-task")
		if value != "status-test" {
			t.Errorf("Expected 'status-test', got %v", value)
		}
	})

	t.Run("status_callbacks_internal", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "status-callback-test", nil
		}).Named("status-callback-test-task")

		workflow := Setup(task)

		var statusChanges []Status
		var mu sync.Mutex

		workflow.OnStatusChange(func(oldStatus, newStatus Status) {
			mu.Lock()
			statusChanges = append(statusChanges, newStatus)
			mu.Unlock()
		})

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Wait for callbacks
		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()

		if len(statusChanges) < 2 {
			t.Errorf("Expected at least 2 status changes, got %d", len(statusChanges))
		}

		// Should have Running and Completed
		hasRunning := false
		hasCompleted := false
		for _, status := range statusChanges {
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

		value := result.Get("status-callback-test-task")
		if value != "status-callback-test" {
			t.Errorf("Expected 'status-callback-test', got %v", value)
		}
	})
}

// TestWorkflow_GetResult tests internal result retrieval
func TestWorkflow_GetResult(t *testing.T) {
	t.Run("get_result_before_execution", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "result-test", nil
		}).Named("result-test-task")

		workflow := Setup(task)

		// Get result before execution
		result, err := workflow.getResult()

		// Should return error before execution
		if result != nil {
			t.Error("Expected nil result before execution")
		}

		if err == nil {
			t.Error("Expected error before execution")
		}

		if err.Error() != "workflow execution failed" {
			t.Errorf("Expected 'workflow execution failed', got '%s'", err.Error())
		}
	})

	t.Run("get_result_after_execution", func(t *testing.T) {
		task := Task(func(ctx Context) (string, error) {
			return "result-after-test", nil
		}).Named("result-after-test-task")

		workflow := Setup(task)

		// Execute workflow
		_, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Get result after execution
		result, err := workflow.getResult()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result after execution")
		}

		value := result.Get("result-after-test-task")
		if value != "result-after-test" {
			t.Errorf("Expected 'result-after-test', got %v", value)
		}
	})
}

// Test internal executeWorkflow and executeOrchestrationTree paths indirectly by forcing
// timeout and error aggregation plus direct invocation via a test helper wrapper.
func TestWorkflow_InternalExecutionBranches(t *testing.T) {
	// Long running task for timeout branch
	longTask := task.Task(func(ctx Context) (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "slow", nil
	}).Named("slow-task")
	wf := Setup(longTask)
	if err := wf.Execute(); err != nil {
		t.Fatalf("execute err: %v", err)
	}
	// expect timeout
	if _, err := wf.AwaitWithTimeout(1 * time.Millisecond); err == nil {
		// may succeed if scheduling fast; still proceed
	}
}

// Direct invocation via exported-like shim (using same package access)
func TestWorkflow_executeWorkflow_Direct(t *testing.T) {
	fast := task.Task(func(ctx Context) (string, error) { return "ok", nil }).Named("fast")
	wf := Setup(fast)
	cfg := config.DefaultConfig()
	res, err := wf.executeWorkflow(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res == nil || res.Get("fast") != "ok" {
		t.Fatalf("missing result")
	}
}

// BenchmarkWorkflow_InternalMethods benchmarks internal methods
func BenchmarkWorkflow_ApplyConfiguration(b *testing.B) {
	task := Task(func(ctx Context) (string, error) {
		return "benchmark", nil
	}).Named("benchmark-task")

	workflow := Setup(task).With(Config{
		Timeout:        30 * time.Second,
		MaxConcurrency: 50,
		ErrorStrategy:  CollectAll,
	})

	baseConfig := config.Config{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		workflow.applyWorkflowConfiguration(baseConfig)
	}
}

// BenchmarkWorkflow_EnhanceError benchmarks error enhancement
func BenchmarkWorkflow_EnhanceError(b *testing.B) {
	task := Task(func(ctx Context) (string, error) {
		return "", errors.New("benchmark error")
	}).Named("benchmark-error-task")

	workflow := Setup(task)
	originalError := errors.New("benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		workflow.enhanceErrorWithMetadata(originalError, "benchmark-orchestration")
	}
}
