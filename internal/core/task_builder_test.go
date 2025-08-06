package core

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
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

	if task.name != "" {
		t.Error("Task name should be empty initially")
	}

	if task.config != nil {
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
	if task.name != "test-task" {
		t.Errorf("Expected name 'test-task', got %q", task.name)
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

	config := Config{
		ErrorStrategy:  CollectAll,
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
	if task.config == nil {
		t.Fatal("Config should not be nil after With()")
	}

	if task.config.ErrorStrategy != CollectAll {
		t.Error("Config ErrorStrategy should be set")
	}

	if task.config.Timeout != 30*time.Second {
		t.Error("Config Timeout should be set")
	}

	if task.config.MaxConcurrency != 50 {
		t.Error("Config MaxConcurrency should be set")
	}

	// Test GetConfig method
	retrievedConfig := task.GetConfig()
	if retrievedConfig == nil {
		t.Fatal("GetConfig() should not return nil")
	}

	if retrievedConfig.ErrorStrategy != CollectAll {
		t.Error("GetConfig() should return correct ErrorStrategy")
	}
}

func TestTaskBuilderErrorBoundary(t *testing.T) {
	task := Task(func() (string, error) {
		return "test", nil
	})

	// Test ErrorBoundary method
	result := task.ErrorBoundary(CollectAll)

	// Should return the same instance (cast back to TaskBuilder to check)
	if result.(*TaskBuilder[string]) != task {
		t.Error("ErrorBoundary() should return the same TaskBuilder instance")
	}

	// Should create config if it doesn't exist
	if task.config == nil {
		t.Fatal("Config should be created by ErrorBoundary()")
	}

	if task.config.ErrorStrategy != CollectAll {
		t.Error("ErrorStrategy should be set by ErrorBoundary()")
	}

	// Test ErrorBoundary with existing config
	task.With(Config{Timeout: 60 * time.Second})
	task.ErrorBoundary(FailFast)

	if task.config.ErrorStrategy != FailFast {
		t.Error("ErrorStrategy should be updated by ErrorBoundary()")
	}

	if task.config.Timeout != 60*time.Second {
		t.Error("Existing config values should be preserved")
	}
}

func TestTaskBuilderFluentAPI(t *testing.T) {
	// Test method chaining
	result := Task(func() (string, error) {
		return "test", nil
	}).Named("chained-task").
		With(Config{Timeout: 30 * time.Second}).
		ErrorBoundary(CollectAll)

	// Cast back to TaskBuilder to access fields
	task := result.(*TaskBuilder[string])

	if task.name != "chained-task" {
		t.Error("Name should be set through chaining")
	}

	if task.config == nil {
		t.Fatal("Config should be set through chaining")
	}

	if task.config.Timeout != 30*time.Second {
		t.Error("Timeout should be set through chaining")
	}

	if task.config.ErrorStrategy != CollectAll {
		t.Error("ErrorStrategy should be set through chaining")
	}
}

func TestTaskBuilderAtomicStatusManagement(t *testing.T) {
	task := Task(func() (string, error) {
		return "test", nil
	})

	// Initial status should be NotStarted
	if status := task.GetStatus(); status != TaskNotStarted {
		t.Errorf("Initial status should be NotStarted, got %v", status)
	}

	// Test compareAndSwapStatus
	if !task.compareAndSwapStatus(TaskNotStarted, TaskRunning) {
		t.Error("compareAndSwapStatus should succeed for valid transition")
	}

	if status := task.GetStatus(); status != TaskRunning {
		t.Errorf("Status should be Running after swap, got %v", status)
	}

	// Test that same swap fails now
	if task.compareAndSwapStatus(TaskNotStarted, TaskRunning) {
		t.Error("compareAndSwapStatus should fail for invalid transition")
	}

	// Test setStatus
	task.setStatus(TaskCompleted)
	if status := task.GetStatus(); status != TaskCompleted {
		t.Errorf("Status should be Completed after setStatus, got %v", status)
	}
}

func TestTaskBuilderSingleExecution(t *testing.T) {
	executionCount := 0
	task := Task(func() (string, error) {
		executionCount++
		return "test", nil
	})

	ctx := context.Background()
	config := DefaultConfig()

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
	expectedError := "task already executed or in progress"
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
				return nil, errors.New("task error")
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
			config := DefaultConfig()

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
	config := Config{
		Timeout: 50 * time.Millisecond, // Shorter than task duration
	}

	result, err := task.Execute(ctx, config)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
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

	config := DefaultConfig()

	result, err := task.Execute(ctx, config)

	if err == nil {
		t.Error("Expected cancellation error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil even on cancellation")
	}

	if !result.HasErrors() {
		t.Error("Result should have errors on cancellation")
	}
}

// Benchmark tests for TaskBuilder
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

	config := Config{
		ErrorStrategy:  CollectAll,
		Timeout:        30 * time.Second,
		MaxConcurrency: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Task(fn).Named("benchmark-task").With(config).ErrorBoundary(FailFast)
	}
}

func BenchmarkTaskExecution(b *testing.B) {
	task := Task(func() (string, error) {
		return "benchmark", nil
	}).Named("benchmark-task")

	ctx := context.Background()
	config := DefaultConfig()

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
	config := DefaultConfig()

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
	config := DefaultConfig()

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
