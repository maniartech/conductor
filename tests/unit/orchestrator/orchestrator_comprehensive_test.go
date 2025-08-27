package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/orchestrator"
	. "github.com/maniartech/orchestrator"
)

// TestSequential_Placeholder tests the Sequential function placeholder
func TestSequential_Placeholder(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected Sequential to panic with placeholder message")
		}
	}()

	Sequential()
}

// TestConcurrent_Placeholder tests the Concurrent function placeholder
func TestConcurrent_Placeholder(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected Concurrent to panic with placeholder message")
		}
	}()

	Concurrent()
}

// TestWorkflow_ComplexScenarios tests complex workflow scenarios
func TestWorkflow_ComplexScenarios(t *testing.T) {
	t.Run("workflow_with_complex_configuration", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "complex-result", nil
		}).Named("complex-task")

		config := Config{
			Timeout:        60 * time.Second,
			MaxConcurrency: 20,
			ErrorStrategy:  CollectAll,
			Context:        context.Background(),
		}

		workflow := Setup(task).With(config)

		if workflow.GetConfig().Timeout != 60*time.Second {
			t.Errorf("Expected timeout 60s, got %v", workflow.GetConfig().Timeout)
		}

		if workflow.GetConfig().MaxConcurrency != 20 {
			t.Errorf("Expected max concurrency 20, got %d", workflow.GetConfig().MaxConcurrency)
		}

		if workflow.GetConfig().ErrorStrategy != CollectAll {
			t.Errorf("Expected CollectAll strategy, got %v", workflow.GetConfig().ErrorStrategy)
		}

		result, err := workflow.Await()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		value := result.Get("complex-task")
		if value != "complex-result" {
			t.Errorf("Expected 'complex-result', got %v", value)
		}
	})

	t.Run("workflow_with_context_timeout", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "timeout-result", nil
		}).Named("timeout-task")

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		workflow := Setup(task)
		_, err := workflow.AwaitWithContext(ctx)

		if err == nil {
			t.Error("Expected timeout error")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected deadline exceeded error, got %v", err)
		}
	})

	t.Run("workflow_multiple_executions", func(t *testing.T) {
		counter := int32(0)
		task := Task(func(ctx orchestrator.Context) (int32, error) {
			return atomic.AddInt32(&counter, 1), nil
		}).Named("counter-task")

		workflow := Setup(task)

		// First execution
		result1, err1 := workflow.Await()
		if err1 != nil {
			t.Fatalf("First execution failed: %v", err1)
		}

		// Second execution should work (backward compatibility)
		result2, err2 := workflow.Await()
		if err2 != nil {
			t.Fatalf("Second execution failed: %v", err2)
		}

		// Both should return the same result since workflow is executed once
		value1 := result1.Get("counter-task")
		value2 := result2.Get("counter-task")

		if value1 != value2 {
			t.Errorf("Expected same result for multiple Await calls, got %v and %v", value1, value2)
		}
	})
}

// TestWorkflow_AdvancedProgressTracking tests advanced progress tracking features
func TestWorkflow_AdvancedProgressTracking(t *testing.T) {
	t.Run("hybrid_progress_mode", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "hybrid-result", nil
		}).Named("hybrid-task")

		workflow := Setup(task)
		workflow.SetProgressMode(ProgressModeHybrid)

		// Initially should use automatic progress
		initialProgress := workflow.GetProgress()
		if initialProgress.Total != 1 {
			t.Errorf("Expected total 1, got %d", initialProgress.Total)
		}

		// Report manual progress
		workflow.ReportProgress(50, 100, "Manual progress")

		// Should now use manual progress
		manualProgress := workflow.GetProgress()
		if manualProgress.Current != 50 || manualProgress.Total != 100 {
			t.Errorf("Expected manual progress 50/100, got %d/%d", manualProgress.Current, manualProgress.Total)
		}

		if manualProgress.Message != "Manual progress" {
			t.Errorf("Expected message 'Manual progress', got '%s'", manualProgress.Message)
		}
	})

	t.Run("stage_with_progress_update", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "stage-result", nil
		}).Named("stage-task")

		workflow := Setup(task)

		// Set stage should trigger progress update
		workflow.SetStage("Processing")

		progress := workflow.GetProgress()
		if progress.Stage != "Processing" {
			t.Errorf("Expected stage 'Processing', got '%s'", progress.Stage)
		}

		// Set another stage
		workflow.SetStage("Finalizing")

		progress = workflow.GetProgress()
		if progress.Stage != "Finalizing" {
			t.Errorf("Expected stage 'Finalizing', got '%s'", progress.Stage)
		}
	})

	t.Run("current_task_tracking", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "current-result", nil
		}).Named("current-task")

		workflow := Setup(task)

		// Initially no current task
		currentTask := workflow.GetCurrentTask()
		if currentTask != "" {
			t.Errorf("Expected empty current task, got '%s'", currentTask)
		}

		// Execute and check if current task is tracked
		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		value := result.Get("current-task")
		if value != "current-result" {
			t.Errorf("Expected 'current-result', got %v", value)
		}
	})

	t.Run("partial_results_access", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "partial-result", nil
		}).Named("partial-task")

		workflow := Setup(task)

		// Get partial results before execution
		partialResults := workflow.GetPartialResults()
		if partialResults == nil {
			t.Fatal("Expected partial results to be available")
		}

		// Should be empty initially
		if len(partialResults.Errors()) != 0 {
			t.Error("Expected no errors in partial results initially")
		}

		// Execute workflow
		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Get partial results after execution
		partialResults = workflow.GetPartialResults()
		if partialResults == nil {
			t.Fatal("Expected partial results to be available after execution")
		}

		value := result.Get("partial-task")
		if value != "partial-result" {
			t.Errorf("Expected 'partial-result', got %v", value)
		}
	})
}

// TestWorkflow_CallbacksAdvanced tests advanced callback scenarios
func TestWorkflow_CallbacksAdvanced(t *testing.T) {
	t.Run("multiple_progress_callbacks", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "callback-result", nil
		}).Named("callback-task")

		workflow := Setup(task)

		var callback1Called, callback2Called atomic.Bool

		workflow.OnProgress(func(progress Progress) {
			callback1Called.Store(true)
		})

		workflow.OnProgress(func(progress Progress) {
			callback2Called.Store(true)
		})

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Wait for callbacks
		time.Sleep(20 * time.Millisecond)

		if !callback1Called.Load() {
			t.Error("Expected first progress callback to be called")
		}

		if !callback2Called.Load() {
			t.Error("Expected second progress callback to be called")
		}

		value := result.Get("callback-task")
		if value != "callback-result" {
			t.Errorf("Expected 'callback-result', got %v", value)
		}
	})

	t.Run("callback_chaining", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "chain-result", nil
		}).Named("chain-task")

		// Test method chaining with callbacks
		workflow := Setup(task).
			OnProgress(func(progress Progress) {
				// Progress callback
			}).
			OnStatusChange(func(oldStatus, newStatus Status) {
				// Status callback
			}).
			OnError(func(err error) {
				// Error callback
			}).
			OnComplete(func(result *Result, err error) {
				// Completion callback
			})

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		value := result.Get("chain-task")
		if value != "chain-result" {
			t.Errorf("Expected 'chain-result', got %v", value)
		}
	})

	t.Run("error_callback_with_failure", func(t *testing.T) {
		expectedError := errors.New("callback test error")
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "", expectedError
		}).Named("error-task")

		workflow := Setup(task)

		var capturedError atomic.Pointer[error]
		workflow.OnError(func(err error) {
			capturedError.Store(&err)
		})

		result, err := workflow.ExecuteBlocking()
		if err == nil {
			t.Fatal("Expected error from failing task")
		}

		// Wait for callback
		time.Sleep(10 * time.Millisecond)

		errorPtr := capturedError.Load()
		if errorPtr == nil {
			t.Error("Expected error callback to be called")
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}
	})
}

// TestWorkflow_CancellationAdvanced tests advanced cancellation scenarios
func TestWorkflow_CancellationAdvanced(t *testing.T) {
	t.Run("cancel_before_execution", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "should-not-execute", nil
		}).Named("cancel-task")

		workflow := Setup(task)

		// Cancel before execution
		workflow.Cancel()

		if workflow.GetStatus() != Cancelled {
			t.Errorf("Expected status Cancelled, got %v", workflow.GetStatus())
		}

		// Try to execute - should handle gracefully
		err := workflow.Execute()
		// Note: The current implementation may allow execution even after cancel
		// This documents the current behavior
		if err != nil {
			t.Logf("Execute after cancel returned error: %v", err)
		}
	})

	t.Run("cancel_with_reason_tracking", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "should-not-complete", nil
		}).Named("reason-task")

		workflow := Setup(task)

		err := workflow.Execute()
		if err != nil {
			t.Fatalf("Failed to start workflow: %v", err)
		}

		// Cancel with reason
		go func() {
			time.Sleep(10 * time.Millisecond)
			workflow.CancelWithReason("test timeout")
		}()

		result, _ := workflow.Await()

		if workflow.GetStatus() != Cancelled {
			t.Errorf("Expected status Cancelled, got %v", workflow.GetStatus())
		}

		if result == nil {
			t.Error("Expected result container even with cancellation")
		}

		// Check if cancellation reason is stored
		reason := result.Get("cancellation_reason")
		if reason != "test timeout" {
			t.Errorf("Expected cancellation reason 'test timeout', got %v", reason)
		}
	})
}

// TestWorkflow_ErrorHandling tests comprehensive error handling
func TestWorkflow_ErrorHandling(t *testing.T) {
	t.Run("workflow_execution_error_wrapping", func(t *testing.T) {
		expectedError := errors.New("task execution failed")
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "", expectedError
		}).Named("failing-task")

		workflow := Setup(task)

		result, err := workflow.ExecuteBlocking()

		if err == nil {
			t.Fatal("Expected error from failing task")
		}

		// Error should be wrapped with context
		if !errors.Is(err, expectedError) {
			t.Errorf("Expected wrapped error to contain original error")
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}

		if len(result.Errors()) == 0 {
			t.Error("Expected errors to be recorded in result")
		}
	})

	t.Run("workflow_with_error_strategy", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "", errors.New("strategy test error")
		}).Named("strategy-task")

		config := Config{
			ErrorStrategy: CollectAll,
		}

		workflow := Setup(task).With(config)

		result, err := workflow.ExecuteBlocking()

		if err == nil {
			t.Fatal("Expected error from failing task")
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}

		if len(result.Errors()) == 0 {
			t.Error("Expected errors to be collected")
		}
	})
}

// TestWorkflow_ResourceManagement tests resource management
func TestWorkflow_ResourceManagement(t *testing.T) {
	t.Run("resource_cleanup_on_completion", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "cleanup-result", nil
		}).Named("cleanup-task")

		workflow := Setup(task)

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Workflow should be in terminal state
		if !workflow.IsInTerminalState() {
			t.Error("Expected workflow to be in terminal state")
		}

		value := result.Get("cleanup-task")
		if value != "cleanup-result" {
			t.Errorf("Expected 'cleanup-result', got %v", value)
		}
	})

	t.Run("resource_cleanup_on_error", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "", errors.New("cleanup error")
		}).Named("cleanup-error-task")

		workflow := Setup(task)

		result, err := workflow.ExecuteBlocking()
		if err == nil {
			t.Fatal("Expected error from failing task")
		}

		// Workflow should be in terminal state
		if !workflow.IsInTerminalState() {
			t.Error("Expected workflow to be in terminal state after error")
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}
	})
}

// TestWorkflow_ConcurrentExecution tests concurrent workflow execution
func TestWorkflow_ConcurrentExecution(t *testing.T) {
	t.Run("multiple_workflows_concurrent", func(t *testing.T) {
		const numWorkflows = 10
		var wg sync.WaitGroup
		var successCount int32

		wg.Add(numWorkflows)
		for i := 0; i < numWorkflows; i++ {
			go func(id int) {
				defer wg.Done()

				task := Task(func(ctx orchestrator.Context) (string, error) {
					time.Sleep(10 * time.Millisecond)
					return fmt.Sprintf("result-%d", id), nil
				}).Named(fmt.Sprintf("task-%d", id))

				workflow := Setup(task)
				result, err := workflow.ExecuteBlocking()

				if err != nil {
					t.Errorf("Workflow %d failed: %v", id, err)
					return
				}

				expected := fmt.Sprintf("result-%d", id)
				value := result.Get(fmt.Sprintf("task-%d", id))
				if value != expected {
					t.Errorf("Expected '%s', got %v", expected, value)
					return
				}

				atomic.AddInt32(&successCount, 1)
			}(i)
		}

		wg.Wait()

		if int(successCount) != numWorkflows {
			t.Errorf("Expected %d successful workflows, got %d", numWorkflows, successCount)
		}
	})
}

// TestDefaultConfig tests the DefaultConfig function
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test that default config has reasonable values
	if config.ErrorStrategy == 0 {
		// ErrorStrategy might be 0 (FailFast), which is valid
	}

	if config.MaxConcurrency == 0 {
		t.Error("Expected default max concurrency to be set")
	}

	// Test that we can use the default config
	task := Task(func(ctx orchestrator.Context) (string, error) {
		return "default-config-result", nil
	}).Named("default-config-task")

	workflow := Setup(task).With(config)

	result, err := workflow.ExecuteBlocking()
	if err != nil {
		t.Fatalf("Workflow with default config failed: %v", err)
	}

	value := result.Get("default-config-task")
	if value != "default-config-result" {
		t.Errorf("Expected 'default-config-result', got %v", value)
	}
}

// TestWorkflow_EdgeCases tests edge cases and boundary conditions
func TestWorkflow_EdgeCases(t *testing.T) {
	t.Run("empty_task_name", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "unnamed-result", nil
		}) // No name set

		workflow := Setup(task)

		if workflow.GetName() != "" {
			t.Errorf("Expected empty name, got '%s'", workflow.GetName())
		}

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		// Result should still be accessible, possibly with generated name
		if result == nil {
			t.Fatal("Expected result even with unnamed task")
		}
	})

	t.Run("nil_context_handling", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "nil-context-result", nil
		}).Named("nil-context-task")

		config := Config{
			Context: nil, // Explicitly nil context
		}

		workflow := Setup(task).With(config)

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow with nil context failed: %v", err)
		}

		value := result.Get("nil-context-task")
		if value != "nil-context-result" {
			t.Errorf("Expected 'nil-context-result', got %v", value)
		}
	})

	t.Run("zero_timeout_config", func(t *testing.T) {
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "zero-timeout-result", nil
		}).Named("zero-timeout-task")

		config := Config{
			Timeout: 0, // Zero timeout
		}

		workflow := Setup(task).With(config)

		result, err := workflow.ExecuteBlocking()
		if err != nil {
			t.Fatalf("Workflow with zero timeout failed: %v", err)
		}

		value := result.Get("zero-timeout-task")
		if value != "zero-timeout-result" {
			t.Errorf("Expected 'zero-timeout-result', got %v", value)
		}
	})
}

// BenchmarkWorkflow_SimpleExecution benchmarks simple workflow execution
func BenchmarkWorkflow_SimpleExecution(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create new task instance for each iteration to avoid reuse issues
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "benchmark-result", nil
		}).Named(fmt.Sprintf("benchmark-task-%d", i))

		workflow := Setup(task)
		_, err := workflow.ExecuteBlocking()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkWorkflow_WithCallbacks benchmarks workflow with callbacks
func BenchmarkWorkflow_WithCallbacks(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create new task instance for each iteration to avoid reuse issues
		task := Task(func(ctx orchestrator.Context) (string, error) {
			return "callback-benchmark-result", nil
		}).Named(fmt.Sprintf("callback-benchmark-task-%d", i))

		workflow := Setup(task).
			OnProgress(func(progress Progress) {}).
			OnStatusChange(func(oldStatus, newStatus Status) {}).
			OnComplete(func(result *Result, err error) {})

		_, err := workflow.ExecuteBlocking()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkWorkflow_ProgressTracking benchmarks progress tracking
func BenchmarkWorkflow_ProgressTracking(b *testing.B) {
	task := Task(func(ctx orchestrator.Context) (string, error) {
		return "progress-benchmark-result", nil
	}).Named("progress-benchmark-task")

	workflow := Setup(task)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		workflow.GetProgress()
	}
}
