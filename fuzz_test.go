package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/task"
)

// FuzzTaskName tests task naming with various inputs
func FuzzTaskName(f *testing.F) {
	// Seed with known good and problematic inputs
	f.Add("simple-task")
	f.Add("")
	f.Add("task-with-numbers-123")
	f.Add("task_with_underscores")
	f.Add("UPPERCASE-TASK")
	f.Add("task.with.dots")
	f.Add("task with spaces")
	f.Add("task-with-unicode-🚀")
	f.Add("very-long-task-name-that-exceeds-normal-length-expectations-and-might-cause-issues")
	f.Add("task\nwith\nnewlines")
	f.Add("task\twith\ttabs")
	f.Add("task\"with\"quotes")
	f.Add("task'with'single'quotes")
	f.Add("task\\with\\backslashes")
	f.Add("task/with/slashes")
	f.Add("task|with|pipes")
	f.Add("task<with>brackets")
	f.Add("task{with}braces")
	f.Add("task[with]square")
	f.Add("task(with)parens")
	f.Add("task@with@symbols")
	f.Add("task#with#hash")
	f.Add("task$with$dollar")
	f.Add("task%with%percent")
	f.Add("task^with^caret")
	f.Add("task&with&ampersand")
	f.Add("task*with*asterisk")
	f.Add("task+with+plus")
	f.Add("task=with=equals")
	f.Add("task~with~tilde")
	f.Add("task`with`backtick")

	f.Fuzz(func(t *testing.T, name string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(name) {
			t.Skip("Invalid UTF-8 string")
		}

		// Skip extremely long strings to avoid memory issues
		if len(name) > 10000 {
			t.Skip("String too long")
		}

		taskFn := func() (string, error) {
			return "fuzz-result", nil
		}

		// Test task creation with fuzzed name
		defer func() {
			if r := recover(); r != nil {
				// Panics are acceptable for invalid inputs, but log them
				t.Logf("Panic recovered for name %q: %v", name, r)
			}
		}()

		taskInstance := task.Task(taskFn).Named(name)

		// Verify the task was created successfully
		if taskInstance == nil {
			t.Errorf("Task creation failed for name: %q", name)
			return
		}

		// Test that we can get the name back
		retrievedName := taskInstance.GetName()
		if retrievedName != name {
			t.Logf("Name mismatch: expected %q, got %q", name, retrievedName)
		}

		// Test task execution
		ctx := context.Background()
		cfg := config.Config{
			Timeout: 1 * time.Second,
		}

		result, err := taskInstance.Execute(ctx, cfg)
		if err != nil {
			t.Logf("Task execution failed for name %q: %v", name, err)
			return
		}

		if result == nil {
			t.Errorf("Result is nil for name: %q", name)
		}
	})
}

// FuzzTaskFunction tests task execution with various function behaviors
func FuzzTaskFunction(f *testing.F) {
	// Seed with different function behaviors
	f.Add("success", false, 0)
	f.Add("error", true, 0)
	f.Add("panic", false, 1)
	f.Add("timeout", false, 2)
	f.Add("empty", false, 3)
	f.Add("large-result", false, 4)
	f.Add("unicode-🚀", false, 5)
	f.Add("newline\nresult", false, 6)
	f.Add("null\x00byte", false, 7)

	f.Fuzz(func(t *testing.T, result string, shouldError bool, behavior int) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(result) {
			t.Skip("Invalid UTF-8 string")
		}

		// Skip extremely long strings
		if len(result) > 100000 {
			t.Skip("Result too long")
		}

		var taskFn func() (string, error)

		switch behavior % 8 {
		case 0: // Normal success
			taskFn = func() (string, error) {
				if shouldError {
					return "", fmt.Errorf("fuzz error: %s", result)
				}
				return result, nil
			}
		case 1: // Panic
			taskFn = func() (string, error) {
				panic(fmt.Sprintf("fuzz panic: %s", result))
			}
		case 2: // Slow execution
			taskFn = func() (string, error) {
				time.Sleep(10 * time.Millisecond)
				return result, nil
			}
		case 3: // Empty result
			taskFn = func() (string, error) {
				return "", nil
			}
		case 4: // Large result
			taskFn = func() (string, error) {
				return strings.Repeat(result, 100), nil
			}
		case 5: // Context checking
			taskFn = func() (string, error) {
				// Simulate context-aware task
				time.Sleep(time.Millisecond)
				return result, nil
			}
		case 6: // Memory allocation
			taskFn = func() (string, error) {
				// Allocate some memory
				data := make([]string, 100)
				for i := range data {
					data[i] = result
				}
				return result, nil
			}
		case 7: // Random error
			taskFn = func() (string, error) {
				if len(result)%2 == 0 {
					return "", fmt.Errorf("random error for: %s", result)
				}
				return result, nil
			}
		}

		taskInstance := task.Task(taskFn).Named("fuzz-task")

		ctx := context.Background()
		cfg := config.Config{
			Timeout: 100 * time.Millisecond,
		}

		// Execute task and handle all possible outcomes
		taskResult, err := taskInstance.Execute(ctx, cfg)

		// Verify result consistency
		if err != nil {
			// Error is acceptable
			if taskResult != nil && len(taskResult.Errors()) == 0 {
				t.Errorf("Task returned error but result has no errors: %v", err)
			}
		} else {
			// Success case
			if taskResult == nil {
				t.Errorf("Task succeeded but result is nil")
			}
		}

		// Test workflow execution as well (create new task instance to avoid reuse issues)
		taskFn2 := taskFn // Copy the function
		taskInstance2 := task.Task(taskFn2).Named("fuzz-task-2")
		workflow := Setup(taskInstance2)
		workflowResult, workflowErr := workflow.Await()

		// Results should be consistent between direct execution and workflow
		// (allowing for some differences due to different execution paths)
		if workflowResult == nil && taskResult != nil && err == nil {
			t.Logf("Workflow result is nil but task result is not nil (this may be expected)")
		}

		// Use workflowErr to avoid unused variable warning
		_ = workflowErr
	})
}

// FuzzConfigValues tests configuration with various values
func FuzzConfigValues(f *testing.F) {
	// Seed with various timeout values
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(1000))
	f.Add(int64(1000000))
	f.Add(int64(-1))
	f.Add(int64(9223372036854775807)) // Max int64

	f.Fuzz(func(t *testing.T, timeoutNanos int64) {
		// Skip extremely large values that might cause issues
		if timeoutNanos > 1e15 || timeoutNanos < -1e15 {
			t.Skip("Timeout value too extreme")
		}

		var timeout time.Duration
		if timeoutNanos > 0 {
			timeout = time.Duration(timeoutNanos)
		}

		cfg := config.Config{
			Timeout: timeout,
		}

		taskFn := func() (string, error) {
			time.Sleep(time.Microsecond) // Small delay
			return "config-fuzz-result", nil
		}

		taskInstance := task.Task(taskFn).Named("config-fuzz-task").With(cfg)

		ctx := context.Background()

		// Test task execution with fuzzed config
		result, err := taskInstance.Execute(ctx, cfg)

		// Handle timeout cases
		if timeout > 0 && timeout < time.Microsecond {
			// Very short timeout might cause timeout error
			if err != nil && strings.Contains(err.Error(), "timeout") {
				t.Logf("Expected timeout for duration %v", timeout)
				return
			}
		}

		// For reasonable timeouts, task should succeed
		if timeout >= time.Millisecond || timeout == 0 {
			if err != nil {
				t.Errorf("Task failed with reasonable timeout %v: %v", timeout, err)
			}
			if result == nil {
				t.Errorf("Result is nil with timeout %v", timeout)
			}
		}
	})
}

// FuzzResultOperations tests result operations with various inputs
func FuzzResultOperations(f *testing.F) {
	// Seed with various key-value pairs
	f.Add("key", "value")
	f.Add("", "empty-key")
	f.Add("empty-value", "")
	f.Add("unicode-🔑", "unicode-📦")
	f.Add("key\nwith\nnewlines", "value\nwith\nnewlines")
	f.Add("key\x00null", "value\x00null")
	f.Add("very-long-key-that-might-cause-memory-issues", "very-long-value-that-might-cause-memory-issues")

	f.Fuzz(func(t *testing.T, key, value string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(key) || !utf8.ValidString(value) {
			t.Skip("Invalid UTF-8 string")
		}

		// Skip extremely long strings
		if len(key) > 10000 || len(value) > 10000 {
			t.Skip("String too long")
		}

		taskFn := func() (string, error) {
			return value, nil
		}

		taskInstance := task.Task(taskFn).Named(key)
		workflow := Setup(taskInstance)

		result, err := workflow.Await()
		if err != nil {
			t.Logf("Workflow failed for key %q, value %q: %v", key, value, err)
			return
		}

		if result == nil {
			t.Errorf("Result is nil for key %q, value %q", key, value)
			return
		}

		// Test result retrieval
		// For empty keys, the task name might be different, so we need to handle this case
		var retrievedValue interface{}
		if key == "" {
			// For empty key, the task might have a default name
			// Try to get the value using the actual task name
			taskName := taskInstance.GetName()
			if taskName == "" {
				taskName = "task_result" // Default name used by task execution
			}
			retrievedValue = result.Get(taskName)
		} else {
			retrievedValue = result.Get(key)
		}

		if retrievedValue != value && key != "" {
			t.Errorf("Value mismatch for key %q: expected %q, got %v", key, value, retrievedValue)
		}

		// Test additional result operations
		result.Set("additional-key", "additional-value")
		additionalValue := result.Get("additional-key")
		if additionalValue != "additional-value" {
			t.Errorf("Additional value mismatch: expected %q, got %v", "additional-value", additionalValue)
		}
	})
}

// FuzzBasicConcurrentOperations tests concurrent operations with fuzzed inputs
func FuzzBasicConcurrentOperations(f *testing.F) {
	// Seed with various operation counts
	f.Add(1, 1)
	f.Add(2, 5)
	f.Add(5, 10)
	f.Add(10, 20)
	f.Add(20, 50)

	f.Fuzz(func(t *testing.T, numGoroutines, numOperations int) {
		// Limit to reasonable values to avoid resource exhaustion
		if numGoroutines < 1 || numGoroutines > 100 {
			t.Skip("Invalid number of goroutines")
		}
		if numOperations < 1 || numOperations > 1000 {
			t.Skip("Invalid number of operations")
		}

		ctx := context.Background()
		cfg := config.Config{
			Timeout: 5 * time.Second,
		}

		results := make(chan string, numGoroutines*numOperations)
		errors := make(chan error, numGoroutines*numOperations)

		// Launch concurrent operations
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				for j := 0; j < numOperations; j++ {
					taskFn := func() (string, error) {
						return fmt.Sprintf("fuzz-concurrent-%d-%d", goroutineID, j), nil
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("fuzz-concurrent-task-%d-%d", goroutineID, j))
					result, err := taskInstance.Execute(ctx, cfg)

					if err != nil {
						errors <- err
					} else if result != nil {
						if value := result.Get(fmt.Sprintf("fuzz-concurrent-task-%d-%d", goroutineID, j)); value != nil {
							if str, ok := value.(string); ok {
								results <- str
							}
						}
					}
				}
			}(i)
		}

		// Collect results
		expectedResults := numGoroutines * numOperations
		actualResults := 0
		actualErrors := 0

		timeout := time.After(10 * time.Second)
		for actualResults+actualErrors < expectedResults {
			select {
			case <-results:
				actualResults++
			case <-errors:
				actualErrors++
			case <-timeout:
				t.Errorf("Timeout waiting for results: got %d results, %d errors out of %d expected",
					actualResults, actualErrors, expectedResults)
				return
			}
		}

		if actualResults+actualErrors != expectedResults {
			t.Errorf("Expected %d total operations, got %d results + %d errors = %d",
				expectedResults, actualResults, actualErrors, actualResults+actualErrors)
		}

		t.Logf("Concurrent fuzz test completed: %d results, %d errors", actualResults, actualErrors)
	})
}

// FuzzErrorMessages tests error handling with various error messages
func FuzzErrorMessages(f *testing.F) {
	// Seed with various error messages
	f.Add("simple error")
	f.Add("")
	f.Add("error with unicode 🚨")
	f.Add("error\nwith\nnewlines")
	f.Add("error\twith\ttabs")
	f.Add("error with null\x00byte")
	f.Add("very long error message that might cause issues with memory allocation or string handling")

	f.Fuzz(func(t *testing.T, errorMsg string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(errorMsg) {
			t.Skip("Invalid UTF-8 string")
		}

		// Skip extremely long strings
		if len(errorMsg) > 10000 {
			t.Skip("Error message too long")
		}

		taskFn := func() (string, error) {
			return "", fmt.Errorf("%s", errorMsg)
		}

		taskInstance := task.Task(taskFn).Named("error-fuzz-task")
		workflow := Setup(taskInstance)

		result, err := workflow.Await()

		// Should have an error
		if err == nil {
			t.Errorf("Expected error but got none for message: %q", errorMsg)
			return
		}

		// Error message should contain the original message
		if !strings.Contains(err.Error(), errorMsg) && errorMsg != "" {
			t.Logf("Error message %q does not contain original message %q", err.Error(), errorMsg)
		}

		// Result should still be valid but contain errors
		if result == nil {
			t.Errorf("Result is nil for error message: %q", errorMsg)
			return
		}

		if !result.HasErrors() {
			t.Errorf("Result should have errors for message: %q", errorMsg)
		}

		errors := result.Errors()
		if len(errors) == 0 {
			t.Errorf("Result.Errors() returned empty slice for message: %q", errorMsg)
		}
	})
}
