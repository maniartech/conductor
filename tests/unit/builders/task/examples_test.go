package task

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/maniartech/orchestrator/internal/errors"
	. "github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
)

// ExampleTaskBuilder_basicUsage demonstrates basic task creation and execution.
func ExampleTaskBuilder_basicUsage() {
	// Create a simple task
	task := Task(func() (string, error) {
		return "Hello, World!", nil
	})

	// Execute the task
	ctx := context.Background()
	cfg := config.DefaultConfig()

	result, err := task.Execute(ctx, cfg)
	if err != nil {
		log.Printf("Task failed: %v", err)
		return
	}

	// Get the result
	value := result.Get("task_result")
	fmt.Printf("Result: %s\n", value)

	// Output:
	// Result: Hello, World!
}

// ExampleTaskBuilder_fluentAPI demonstrates the fluent API for task configuration.
func ExampleTaskBuilder_fluentAPI() {
	// Create a task with fluent configuration
	task := Task(func() (int, error) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		return 42, nil
	}).Named("answer-task").
		With(config.Config{
			Timeout: 30 * time.Second,
		}).
		ErrorBoundary(errors.CollectAll)

	ctx := context.Background()
	cfg := config.DefaultConfig()

	result, err := task.Execute(ctx, cfg)
	if err != nil {
		log.Printf("Task failed: %v", err)
		return
	}

	// Get the named result
	value := result.Get("answer-task")
	fmt.Printf("Answer: %d\n", value)

	// Output:
	// Answer: 42
}

// ExampleTaskBuilder_genericTypes demonstrates generic type support.
func ExampleTaskBuilder_genericTypes() {
	// String task
	stringTask := Task(func() (string, error) {
		return "text result", nil
	})

	// Integer task
	intTask := Task(func() (int, error) {
		return 123, nil
	})

	// Custom struct task
	type User struct {
		ID   int
		Name string
	}

	userTask := Task(func() (User, error) {
		return User{ID: 1, Name: "John Doe"}, nil
	})

	ctx := context.Background()
	cfg := config.DefaultConfig()

	// Execute all tasks
	stringResult, _ := stringTask.Execute(ctx, cfg)
	intResult, _ := intTask.Execute(ctx, cfg)
	userResult, _ := userTask.Execute(ctx, cfg)

	fmt.Printf("String: %s\n", stringResult.Get("task_result"))
	fmt.Printf("Integer: %d\n", intResult.Get("task_result"))
	fmt.Printf("User: %+v\n", userResult.Get("task_result"))

	// Output:
	// String: text result
	// Integer: 123
	// User: {ID:1 Name:John Doe}
}

// ExampleTaskBuilder_errorHandling demonstrates comprehensive error handling.
func ExampleTaskBuilder_errorHandling() {
	// Task that returns an error
	errorTask := Task(func() (string, error) {
		return "", fmt.Errorf("something went wrong")
	}).Named("error-task")

	// Task that panics
	panicTask := Task(func() (string, error) {
		panic("unexpected panic")
	}).Named("panic-task")

	ctx := context.Background()
	cfg := config.DefaultConfig()

	// Execute error task
	result1, err1 := errorTask.Execute(ctx, cfg)
	fmt.Printf("Error task - Error: %v, HasErrors: %t\n", err1 != nil, result1.HasErrors())

	// Execute panic task
	result2, err2 := panicTask.Execute(ctx, cfg)
	fmt.Printf("Panic task - Error: %v, HasErrors: %t\n", err2 != nil, result2.HasErrors())

	// Check error details
	if result2.HasErrors() {
		errs := result2.Errors()
		fmt.Printf("Error count: %d\n", len(errs))
		fmt.Printf("Has stack trace: %t\n", len(errs[0].Stack) > 0)
	}

	// Output:
	// Error task - Error: true, HasErrors: true
	// Panic task - Error: true, HasErrors: true
	// Error count: 1
	// Has stack trace: true
}

// ExampleTaskBuilder_timeoutHandling demonstrates timeout handling.
func ExampleTaskBuilder_timeoutHandling() {
	// Task that takes longer than timeout
	slowTask := Task(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "completed", nil
	})

	// Configure with fluent API
	configured := slowTask.Named("slow-task").
		With(config.Config{
			Timeout: 50 * time.Millisecond, // Shorter than task duration
		})

	ctx := context.Background()
	cfg := config.DefaultConfig()

	start := time.Now()
	result, err := configured.Execute(ctx, cfg)
	duration := time.Since(start)

	fmt.Printf("Timed out: %t\n", err != nil)
	fmt.Printf("Duration < 80ms: %t\n", duration < 80*time.Millisecond)
	fmt.Printf("Task status: %s\n", slowTask.GetStatus())
	fmt.Printf("Result has errors: %t\n", result.HasErrors())

	// Output:
	// Timed out: true
	// Duration < 80ms: true
	// Task status: Cancelled
	// Result has errors: true
}

// ExampleTaskBuilder_cancellation demonstrates context cancellation.
func ExampleTaskBuilder_cancellation() {
	task := Task(func() (string, error) {
		return "completed", nil
	})
	namedTask := task.Named("cancellable-task")

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := config.DefaultConfig()

	result, err := namedTask.Execute(ctx, cfg)

	fmt.Printf("Cancelled: %t\n", err != nil)
	fmt.Printf("Task status: %s\n", namedTask.GetStatus())
	fmt.Printf("Result has errors: %t\n", result.HasErrors())

	// Output:
	// Cancelled: true
	// Task status: Cancelled
	// Result has errors: true
}

// ExampleTaskBuilder_statusTracking demonstrates status tracking throughout execution.
func ExampleTaskBuilder_statusTracking() {
	task := Task(func() (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "done", nil
	})
	namedTask := task.Named("status-task")

	fmt.Printf("Initial status: %s\n", namedTask.GetStatus())

	// Execute in background to observe status changes
	ctx := context.Background()
	cfg := config.DefaultConfig()

	go func() {
		namedTask.Execute(ctx, cfg)
	}()

	// Give it a moment to start
	time.Sleep(5 * time.Millisecond)
	fmt.Printf("During execution: %s\n", namedTask.GetStatus())

	// Wait for completion
	time.Sleep(20 * time.Millisecond)
	fmt.Printf("Final status: %s\n", namedTask.GetStatus())

	// Output:
	// Initial status: NotStarted
	// During execution: Running
	// Final status: Completed
}

// ExampleTaskBuilder_configurationInheritance demonstrates configuration inheritance.
func ExampleTaskBuilder_configurationInheritance() {
	// Global configuration
	globalConfig := config.Config{
		Timeout:        60 * time.Second,
		MaxConcurrency: 10,
		ErrorStrategy:  errors.FailFast,
	}

	// Task with local configuration override
	task := Task(func() (string, error) {
		return "configured", nil
	})

	task.Named("config-task").
		With(config.Config{
			Timeout: 30 * time.Second, // Override global timeout
			// MaxConcurrency and ErrorStrategy inherited from global
		})

	ctx := context.Background()

	result, err := task.Execute(ctx, globalConfig)
	if err != nil {
		log.Printf("Task failed: %v", err)
		return
	}

	// Check final configuration used
	finalConfig := task.GetConfig()
	fmt.Printf("Task timeout: %s\n", finalConfig.Timeout)
	fmt.Printf("Result: %s\n", result.Get("config-task"))

	// Output:
	// Task timeout: 30s
	// Result: configured
}

// ExampleTaskBuilder_performanceCharacteristics demonstrates performance features.
func ExampleTaskBuilder_performanceCharacteristics() {
	task := Task(func() (int, error) {
		return 42, nil
	}).Named("perf-task")

	// Status operations are zero-allocation and thread-safe
	fmt.Printf("Initial status: %s\n", task.GetStatus())

	ctx := context.Background()
	cfg := config.DefaultConfig()

	result, err := task.Execute(ctx, cfg)
	if err != nil {
		log.Printf("Task failed: %v", err)
		return
	}

	fmt.Printf("Final result: %d\n", result.Get("perf-task"))
	fmt.Printf("Final status: %s\n", task.GetStatus())

	// Output:
	// Initial status: NotStarted
	// Final result: 42
	// Final status: Completed
}
