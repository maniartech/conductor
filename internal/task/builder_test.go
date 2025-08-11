package task

import (
	"context"
	systemErrors "errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/types"
)

func TestTask(t *testing.T) {
	// Test basic task creation
	task := Task(func() (string, error) {
		return "hello", nil
	})

	if task == nil {
		t.Fatal("Task() returned nil")
	}

	if task.fn == nil {
		t.Error("Task function should not be nil")
	}

	if task.GetName() != "" {
		t.Error("Task name should be empty initially")
	}

	if task.GetConfig() != nil {
		t.Error("Task config should be nil initially")
	}
}

func TestTaskPanicOnNilFunction(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Task() should panic when function is nil")
		}
	}()

	Task[string](nil)
}

func TestTaskBuilderGenericTypes(t *testing.T) {
	// Test different generic types
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "string type",
			test: func(t *testing.T) {
				task := Task(func() (string, error) {
					return "test", nil
				})
				if task == nil {
					t.Error("String task should not be nil")
				}
			},
		},
		{
			name: "int type",
			test: func(t *testing.T) {
				task := Task(func() (int, error) {
					return 42, nil
				})
				if task == nil {
					t.Error("Int task should not be nil")
				}
			},
		},
		{
			name: "bool type",
			test: func(t *testing.T) {
				task := Task(func() (bool, error) {
					return true, nil
				})
				if task == nil {
					t.Error("Bool task should not be nil")
				}
			},
		},
		{
			name: "slice type",
			test: func(t *testing.T) {
				task := Task(func() ([]string, error) {
					return []string{"a", "b", "c"}, nil
				})
				if task == nil {
					t.Error("Slice task should not be nil")
				}
			},
		},
		{
			name: "map type",
			test: func(t *testing.T) {
				task := Task(func() (map[string]int, error) {
					return map[string]int{"key": 42}, nil
				})
				if task == nil {
					t.Error("Map task should not be nil")
				}
			},
		},
		{
			name: "struct type",
			test: func(t *testing.T) {
				type User struct {
					ID   int
					Name string
				}
				task := Task(func() (User, error) {
					return User{ID: 123, Name: "John"}, nil
				})
				if task == nil {
					t.Error("Struct task should not be nil")
				}
			},
		},
		{
			name: "pointer type",
			test: func(t *testing.T) {
				task := Task(func() (*string, error) {
					s := "test"
					return &s, nil
				})
				if task == nil {
					t.Error("Pointer task should not be nil")
				}
			},
		},
		{
			name: "interface type",
			test: func(t *testing.T) {
				task := Task(func() (interface{}, error) {
					return "anything", nil
				})
				if task == nil {
					t.Error("Interface task should not be nil")
				}
			},
		},
		{
			name: "channel type",
			test: func(t *testing.T) {
				task := Task(func() (chan int, error) {
					ch := make(chan int, 1)
					ch <- 42
					return ch, nil
				})
				if task == nil {
					t.Error("Channel task should not be nil")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, test.test)
	}
}

func TestTaskBuilderNamed(t *testing.T) {
	task := Task(func() (string, error) {
		return "test", nil
	})

	// Test Named method
	result := task.Named("test-task")

	// Should return the same instance (cast back to TaskBuilder to check)
	if result.(*TaskBuilder[string]) != task {
		t.Error("Named() should return the same TaskBuilder instance")
	}

	// Should set the name
	if task.GetName() != "test-task" {
		t.Errorf("Expected name 'test-task', got %q", task.GetName())
	}

	// Test GetName method
	if task.GetName() != "test-task" {
		t.Errorf("GetName() expected 'test-task', got %q", task.GetName())
	}
}

func TestTaskBuilderWith(t *testing.T) {
	task := Task(func() (string, error) {
		return "test", nil
	})

	config := config.Config{
		ErrorStrategy:  errors.CollectAll,
		Timeout:        30 * time.Second,
		MaxConcurrency: 50,
	}

	// Test With method
	result := task.With(config)

	// Should return the same instance (cast back to TaskBuilder to check)
	if result.(*TaskBuilder[string]) != task {
		t.Error("With() should return the same TaskBuilder instance")
	}

	// Should set the config
	if task.GetConfig() == nil {
		t.Fatal("config.Config should not be nil after With()")
	}

	if task.GetConfig().ErrorStrategy != errors.CollectAll {
		t.Error("config.Config ErrorStrategy should be set")
	}

	if task.GetConfig().Timeout != 30*time.Second {
		t.Error("config.Config Timeout should be set")
	}

	if task.GetConfig().MaxConcurrency != 50 {
		t.Error("config.Config MaxConcurrency should be set")
	}

	// Test GetConfig method
	retrievedConfig := task.GetConfig()
	if retrievedConfig == nil {
		t.Fatal("GetConfig() should not return nil")
	}

	if retrievedConfig.ErrorStrategy != errors.CollectAll {
		t.Error("GetConfig() should return correct ErrorStrategy")
	}
}

func TestTaskBuilderErrorBoundary(t *testing.T) {
	task := Task(func() (string, error) {
		return "test", nil
	})

	// Test ErrorBoundary method
	result := task.ErrorBoundary(errors.CollectAll)

	// Should return the same instance (cast back to TaskBuilder to check)
	if result.(*TaskBuilder[string]) != task {
		t.Error("ErrorBoundary() should return the same TaskBuilder instance")
	}

	// Should create config if it doesn't exist
	if task.GetConfig() == nil {
		t.Fatal("config.Config should be created by ErrorBoundary()")
	}

	if task.GetConfig().ErrorStrategy != errors.CollectAll {
		t.Error("ErrorStrategy should be set by ErrorBoundary()")
	}

	// Test ErrorBoundary with existing config
	task.With(config.Config{Timeout: 60 * time.Second})
	task.ErrorBoundary(errors.FailFast)

	if task.GetConfig().ErrorStrategy != errors.FailFast {
		t.Error("ErrorStrategy should be updated by ErrorBoundary()")
	}

	if task.GetConfig().Timeout != 60*time.Second {
		t.Error("Existing config values should be preserved")
	}
}

func TestTaskBuilderFluentAPI(t *testing.T) {
	// Test method chaining
	result := Task(func() (string, error) {
		return "test", nil
	}).Named("chained-task").
		With(config.Config{Timeout: 30 * time.Second}).
		ErrorBoundary(errors.CollectAll)

	// Cast back to TaskBuilder to access fields
	task := result.(*TaskBuilder[string])

	if task.GetName() != "chained-task" {
		t.Error("Name should be set through chaining")
	}

	if task.GetConfig() == nil {
		t.Fatal("config.Config should be set through chaining")
	}

	if task.GetConfig().Timeout != 30*time.Second {
		t.Error("Timeout should be set through chaining")
	}

	if task.GetConfig().ErrorStrategy != errors.CollectAll {
		t.Error("ErrorStrategy should be set through chaining")
	}
}

func TestTaskBuilderAtomicStatusManagement(t *testing.T) {
	task := Task(func() (string, error) {
		return "test", nil
	})

	// Initial status should be NotStarted
	if status := task.GetStatus(); status != orchestration.NotStarted {
		t.Errorf("Initial status should be NotStarted, got %v", status)
	}

	// Test CompareAndSwapStatus
	if !task.CompareAndSwapStatus(types.NotStarted, types.Running) {
		t.Error("CompareAndSwapStatus should succeed for valid transition")
	}

	if status := task.GetStatus(); status != types.Running {
		t.Errorf("Status should be Running after swap, got %v", status)
	}

	// Test that same swap fails now
	if task.CompareAndSwapStatus(types.NotStarted, types.Running) {
		t.Error("CompareAndSwapStatus should fail for invalid transition")
	}

	// Test SetStatus
	task.SetStatus(types.Completed)
	if status := task.GetStatus(); status != types.Completed {
		t.Errorf("Status should be Completed after SetStatus, got %v", status)
	}
}

func TestTaskBuilderSingleExecution(t *testing.T) {
	executionCount := 0
	task := Task(func() (string, error) {
		executionCount++
		return "test", nil
	})

	ctx := context.Background()
	config := config.DefaultConfig()

	// First execution should succeed
	result1, err1 := task.Execute(ctx, config)
	if err1 != nil {
		t.Errorf("First execution should succeed, got error: %v", err1)
	}
	if result1 == nil {
		t.Fatal("First execution result should not be nil")
	}

	// Second execution should fail
	result2, err2 := task.Execute(ctx, config)
	if err2 == nil {
		t.Error("Second execution should fail")
	}
	if result2 != nil {
		t.Error("Second execution result should be nil")
	}

	// Function should only be called once
	if executionCount != 1 {
		t.Errorf("Function should be called exactly once, got %d", executionCount)
	}

	// Check error message
	expectedError := "task orchestration already executed or in progress"
	if !contains(err2.Error(), expectedError) {
		t.Errorf("Expected error to contain %q, got %q", expectedError, err2.Error())
	}
}

func TestTaskBuilderExecute(t *testing.T) {
	tests := []struct {
		name           string
		taskFunc       func() (interface{}, error)
		expectedResult interface{}
		expectedError  bool
		taskName       string
	}{
		{
			name: "successful execution",
			taskFunc: func() (interface{}, error) {
				return "success", nil
			},
			expectedResult: "success",
			expectedError:  false,
			taskName:       "success-task",
		},
		{
			name: "execution with error",
			taskFunc: func() (interface{}, error) {
				return nil, systemErrors.New("task error")
			},
			expectedResult: nil,
			expectedError:  true,
			taskName:       "error-task",
		},
		{
			name: "execution with panic",
			taskFunc: func() (interface{}, error) {
				panic("task panic")
			},
			expectedResult: nil,
			expectedError:  true,
			taskName:       "panic-task",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := Task(test.taskFunc).Named(test.taskName)
			ctx := context.Background()
			config := config.DefaultConfig()

			result, err := task.Execute(ctx, config)

			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if result == nil {
					t.Fatal("Result should not be nil even on error")
				}
				if !result.HasErrors() {
					t.Error("Result should have errors")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result == nil {
					t.Fatal("Result should not be nil")
				}
				if result.HasErrors() {
					t.Error("Result should not have errors")
				}

				// Check result value
				if test.expectedResult != nil {
					value := result.Get(test.taskName)
					if value != test.expectedResult {
						t.Errorf("Expected result %v, got %v", test.expectedResult, value)
					}
				}
			}
		})
	}
}

func TestTaskBuilderExecuteWithTimeout(t *testing.T) {
	// Test timeout handling
	task := Task(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "completed", nil
	}).Named("timeout-task")

	ctx := context.Background()
	config := config.Config{
		Timeout: 50 * time.Millisecond, // Shorter than task duration
	}

	result, err := task.Execute(ctx, config)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if !systemErrors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil even on timeout")
	}

	if !result.HasErrors() {
		t.Error("Result should have errors on timeout")
	}
}

func TestTaskBuilderExecuteWithCancellation(t *testing.T) {
	// Test context cancellation by cancelling before task starts
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	task := Task(func() (string, error) {
		return "completed", nil
	}).Named("cancel-task")

	config := config.DefaultConfig()

	result, err := task.Execute(ctx, config)

	if err == nil {
		t.Error("Expected cancellation error")
	}

	if !systemErrors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil even on cancellation")
	}

	if !result.HasErrors() {
		t.Error("Result should have errors on cancellation")
	}
}

// Benchmark tests for TaskBuilder creation and fluent API performance
func BenchmarkTaskCreation(b *testing.B) {
	fn := func() (string, error) {
		return "test", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Task(fn)
	}
}

func BenchmarkTaskBuilderFluentAPI(b *testing.B) {
	fn := func() (string, error) {
		return "test", nil
	}

	config := config.Config{
		ErrorStrategy:  errors.CollectAll,
		Timeout:        30 * time.Second,
		MaxConcurrency: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Task(fn).Named("benchmark-task").With(config).ErrorBoundary(errors.FailFast)
	}
}

func BenchmarkTaskExecution(b *testing.B) {
	task := Task(func() (string, error) {
		return "benchmark", nil
	}).Named("benchmark-task")

	ctx := context.Background()
	config := config.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task.Execute(ctx, config)
	}
}

// Example tests for documentation
func ExampleTask() {
	// Create a simple string task
	task := Task(func() (string, error) {
		return "Hello, World!", nil
	})

	ctx := context.Background()
	config := config.DefaultConfig()

	result, err := task.Execute(ctx, config)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	value := result.Get("task_result")
	fmt.Printf("Result: %v\n", value)

	// Output:
	// Result: Hello, World!
}

func ExampleTaskBuilder_Named() {
	task := Task(func() (int, error) {
		return 42, nil
	}).Named("answer-task")

	ctx := context.Background()
	config := config.DefaultConfig()

	result, err := task.Execute(ctx, config)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	value := result.Get("answer-task")
	fmt.Printf("Answer: %v\n", value)

	// Output:
	// Answer: 42
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Race condition tests for concurrent status access during task execution
func TestTaskBuilderConcurrentStatusAccess(t *testing.T) {
	task := Task(func() (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "test", nil
	})

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Multiple goroutines trying to read status concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			status := task.GetStatus()
			// Status should be one of the valid values
			if status != orchestration.NotStarted && status != orchestration.Running && status != orchestration.Completed && status != orchestration.Cancelled {
				t.Errorf("Invalid status: %v", status)
			}
		}()
	}

	// Start execution in background
	go func() {
		ctx := context.Background()
		config := config.DefaultConfig()
		task.Execute(ctx, config)
	}()

	wg.Wait()
}

func TestTaskBuilderPanicRecoveryWithStackTrace(t *testing.T) {
	task := Task(func() (string, error) {
		panic("test panic for stack trace")
	})
	task.Named("panic-task")

	ctx := context.Background()
	cfg := config.DefaultConfig()

	result, err := task.Execute(ctx, cfg)

	// Should recover from panic
	if err == nil {
		t.Error("Expected error from panic recovery")
	}

	// Should contain panic message
	if !contains(err.Error(), "test panic for stack trace") {
		t.Errorf("Error should contain panic message, got: %v", err)
	}

	// Should contain stack trace
	if !contains(err.Error(), "Stack trace:") {
		t.Errorf("Error should contain stack trace, got: %v", err)
	}

	// Result should have errors
	if result == nil {
		t.Fatal("Result should not be nil even on panic")
	}

	if !result.HasErrors() {
		t.Error("Result should have errors on panic")
	}

	// Check OperationError has stack trace
	errors := result.Errors()
	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}

	if len(errors[0].Stack) == 0 {
		t.Error("OperationError should have stack trace")
	}

	// Task status should be Completed (even with panic)
	if status := task.GetStatus(); status != orchestration.Completed {
		t.Errorf("Task status should be Completed after panic, got %v", status)
	}
}

func TestTaskBuilderTimeoutHandling(t *testing.T) {
	task := Task(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "completed", nil
	})
	task.Named("timeout-task")

	ctx := context.Background()
	cfg := config.Config{
		Timeout: 50 * time.Millisecond, // Shorter than task duration
	}

	start := time.Now()
	result, err := task.Execute(ctx, cfg)
	duration := time.Since(start)

	// Should timeout
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !systemErrors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	// Should timeout quickly (within reasonable bounds)
	if duration > 80*time.Millisecond {
		t.Errorf("Timeout took too long: %v", duration)
	}

	// Task status should be Cancelled
	if status := task.GetStatus(); status != orchestration.Cancelled {
		t.Errorf("Task status should be Cancelled after timeout, got %v", status)
	}

	// Result should have errors with timing information
	if result == nil {
		t.Fatal("Result should not be nil even on timeout")
	}

	errors := result.Errors()
	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}

	if errors[0].Duration <= 0 {
		t.Error("OperationError should have positive duration")
	}
}

func TestTaskBuilderCancellationHandling(t *testing.T) {
	task := Task(func() (string, error) {
		return "completed", nil
	})
	task.Named("cancel-task")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := config.DefaultConfig()

	result, err := task.Execute(ctx, cfg)

	// Should be cancelled
	if err == nil {
		t.Error("Expected cancellation error")
	}

	if !systemErrors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	// Task status should be Cancelled
	if status := task.GetStatus(); status != orchestration.Cancelled {
		t.Errorf("Task status should be Cancelled, got %v", status)
	}

	// Result should have errors
	if result == nil {
		t.Fatal("Result should not be nil even on cancellation")
	}

	if !result.HasErrors() {
		t.Error("Result should have errors on cancellation")
	}
}

func TestTaskBuilderSafeExecute(t *testing.T) {
	tests := []struct {
		name        string
		taskFunc    func() (string, error)
		expectError bool
		expectPanic bool
	}{
		{
			name: "successful execution",
			taskFunc: func() (string, error) {
				return "success", nil
			},
			expectError: false,
			expectPanic: false,
		},
		{
			name: "execution with error",
			taskFunc: func() (string, error) {
				return "", systemErrors.New("task error")
			},
			expectError: true,
			expectPanic: false,
		},
		{
			name: "execution with panic",
			taskFunc: func() (string, error) {
				panic("task panic")
			},
			expectError: true,
			expectPanic: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := Task(test.taskFunc)
			ctx := context.Background()

			result, err := task.safeExecute(ctx)

			if test.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if test.expectPanic && !contains(err.Error(), "panic recovered") {
					t.Error("Expected panic recovery message")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result != "success" {
					t.Errorf("Expected 'success', got %v", result)
				}
			}
		})
	}
}

func TestTaskBuilderCaptureStack(t *testing.T) {
	task := Task(func() (string, error) {
		return "test", nil
	})

	stack := task.captureStack()

	if len(stack) == 0 {
		t.Error("Stack trace should not be empty")
	}

	// Stack trace should contain function names
	stackStr := string(stack)
	if !contains(stackStr, "TestTaskBuilderCaptureStack") {
		t.Error("Stack trace should contain test function name")
	}
}

func TestTaskBuilderErrorMetadata(t *testing.T) {
	task := Task(func() (string, error) {
		time.Sleep(1 * time.Millisecond) // Ensure some duration
		return "", systemErrors.New("test error")
	})
	task.Named("metadata-task")

	ctx := context.Background()
	cfg := config.DefaultConfig()

	start := time.Now()
	result, err := task.Execute(ctx, cfg)
	end := time.Now()

	if err == nil {
		t.Fatal("Expected error")
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	errors := result.Errors()
	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}

	opErr := errors[0]

	// Check error metadata
	if opErr.Error.Error() != "test error" {
		t.Errorf("Expected 'test error', got %q", opErr.Error.Error())
	}

	if opErr.Index != 0 {
		t.Errorf("Expected Index 0, got %d", opErr.Index)
	}

	if opErr.Duration <= 0 {
		t.Error("Duration should be positive")
	}

	if opErr.Timestamp.Before(start) || opErr.Timestamp.After(end) {
		t.Error("Timestamp should be within execution window")
	}

	if opErr.OpID != "task-metadata-task" {
		t.Errorf("Expected OpID 'task-metadata-task', got %q", opErr.OpID)
	}

	if len(opErr.Stack) == 0 {
		t.Error("Stack should not be empty")
	}
}

// Stress tests for concurrent execution
func TestTaskBuilderStressConcurrentExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const numTasks = 1000
	const concurrency = 100

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			task := Task(func() (int, error) {
				// Simulate some work
				time.Sleep(time.Microsecond * time.Duration(id%10))
				return id, nil
			})
			task.Named(fmt.Sprintf("stress-task-%d", id))

			ctx := context.Background()
			cfg := config.DefaultConfig()

			result, err := task.Execute(ctx, cfg)

			if err != nil {
				t.Errorf("Task %d failed: %v", id, err)
				return
			}

			if result == nil {
				t.Errorf("Task %d returned nil result", id)
				return
			}

			if value, ok := result.Get(fmt.Sprintf("stress-task-%d", id)).(int); !ok || value != id {
				t.Errorf("Task %d returned wrong value: expected %d, got %v", id, id, value)
			}
		}(i)
	}

	wg.Wait()
}

func TestTaskBuilderStressErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const numTasks = 500
	var wg sync.WaitGroup

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			var task *TaskBuilder[string]
			if id%3 == 0 {
				// Panic task
				task = Task(func() (string, error) {
					panic(fmt.Sprintf("panic-%d", id))
				})
			} else if id%3 == 1 {
				// Error task
				task = Task(func() (string, error) {
					return "", fmt.Errorf("error-%d", id)
				})
			} else {
				// Success task
				task = Task(func() (string, error) {
					return fmt.Sprintf("success-%d", id), nil
				})
			}

			task.Named(fmt.Sprintf("stress-error-task-%d", id))

			ctx := context.Background()
			cfg := config.DefaultConfig()

			result, err := task.Execute(ctx, cfg)

			// All tasks should return a result (even failed ones)
			if result == nil {
				t.Errorf("Task %d returned nil result", id)
				return
			}

			// Check expected behavior based on task type
			if id%3 == 2 { // Success task
				if err != nil {
					t.Errorf("Success task %d should not have error: %v", id, err)
				}
				if result.HasErrors() {
					t.Errorf("Success task %d should not have errors in result", id)
				}
			} else { // Error or panic task
				if err == nil {
					t.Errorf("Error/panic task %d should have error", id)
				}
				if !result.HasErrors() {
					t.Errorf("Error/panic task %d should have errors in result", id)
				}
			}
		}(i)
	}

	wg.Wait()
}

// Benchmark tests for execution engine performance
func BenchmarkTaskBuilderSafeExecute(b *testing.B) {
	task := Task(func() (string, error) {
		return "benchmark", nil
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task.safeExecute(ctx)
	}
}

func BenchmarkTaskBuilderStatusOperations(b *testing.B) {
	task := Task(func() (string, error) {
		return "test", nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task.GetStatus()
		task.SetStatus(types.Running)
		task.CompareAndSwapStatus(types.Running, types.Completed)
	}
}

func BenchmarkTaskBuilderStackCapture(b *testing.B) {
	task := Task(func() (string, error) {
		return "test", nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task.captureStack()
	}
}

func BenchmarkTaskBuilderConcurrentExecution(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := Task(func() (string, error) {
				return "benchmark", nil
			})

			ctx := context.Background()
			cfg := config.DefaultConfig()

			task.Execute(ctx, cfg)
		}
	})
}

// Example tests for execution engine documentation
func ExampleTaskBuilder_GetStatus() {
	task := Task(func() (string, error) {
		return "Hello, World!", nil
	})

	fmt.Printf("Initial status: %v\n", task.GetStatus())

	ctx := context.Background()
	cfg := config.DefaultConfig()
	task.Execute(ctx, cfg)

	fmt.Printf("Final status: %v\n", task.GetStatus())

	// Output:
	// Initial status: NotStarted
	// Final status: Completed
}

func ExampleTaskBuilder_safeExecute() {
	task := Task(func() (string, error) {
		return "Safe execution", nil
	})

	ctx := context.Background()
	result, err := task.safeExecute(ctx)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n", result)

	// Output:
	// Result: Safe execution
}

func TestTask_PathResolutionCoverage(t *testing.T) {
	// simple task
	tt := Task(func() (string, error) { return "x", nil }).Named("path-task")

	// GetCurrentPath should return an operation id containing type prefix
	path := tt.GetCurrentPath()
	if path == "" {
		t.Fatalf("expected non-empty current path")
	}

	// GetByPath success
	got, err := tt.GetByPath(path)
	if err != nil || got == nil {
		t.Fatalf("expected to retrieve task by path, err=%v", err)
	}

	// GetByPath not found
	_, err = tt.GetByPath(path + "-missing")
	if err == nil {
		t.Fatalf("expected error for missing path")
	}

	// ListAllPaths contains exactly the current path
	paths := tt.ListAllPaths()
	if len(paths) != 1 || paths[0] != path {
		t.Fatalf("expected single path %s, got %v", path, paths)
	}

	// FindByName matches
	matches := tt.FindByName("path-task")
	if len(matches) != 1 || matches[0].Path != path {
		t.Fatalf("expected match for name path-task, got %v", matches)
	}

	// Orchestration tree root
	tree := tt.GetOrchestrationTree()
	if tree == nil || tree.Path != path || len(tree.Children) != 0 {
		t.Fatalf("unexpected tree: %+v", tree)
	}

	// Query API basic usage
	q := tt.Query()
	if q == nil {
		t.Fatalf("expected non-nil query")
	}
	leaf := q.FindLeafNodes()
	if len(leaf) == 0 {
		t.Fatalf("expected at least one leaf node")
	}
}
