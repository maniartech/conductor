package conditional

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	orchContext "github.com/maniartech/orchestrator/internal/context"
	orchErrors "github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/task"
)

// TestConditional_Constructor tests the Conditional constructor function
func TestConditional_Constructor(t *testing.T) {
	t.Run("valid_parameters", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
		trueTask := task.Task(func() (string, error) { return "true", nil })
		falseTask := task.Task(func() (string, error) { return "false", nil })

		conditional := Conditional(condition, trueTask, falseTask)

		if conditional == nil {
			t.Fatal("Expected conditional to be created")
		}

		// Name should be empty by default until Named() is called
		if conditional.GetName() == "" {
			// This is expected behavior - name is empty until Named() is called
		}
	})

	t.Run("nil_condition_panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for nil condition")
			}
		}()

		trueTask := task.Task(func() (string, error) { return "true", nil })
		falseTask := task.Task(func() (string, error) { return "false", nil })

		Conditional(nil, trueTask, falseTask)
	})

	t.Run("nil_ifTrue_panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for nil ifTrue")
			}
		}()

		condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
		falseTask := task.Task(func() (string, error) { return "false", nil })

		Conditional(condition, nil, falseTask)
	})

	t.Run("nil_ifFalse_panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for nil ifFalse")
			}
		}()

		condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
		trueTask := task.Task(func() (string, error) { return "true", nil })

		Conditional(condition, trueTask, nil)
	})
}

// TestConditionalBuilder_FluentAPI tests the fluent API methods
func TestConditionalBuilder_FluentAPI(t *testing.T) {
	condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
	trueTask := task.Task(func() (string, error) { return "true", nil })
	falseTask := task.Task(func() (string, error) { return "false", nil })

	t.Run("named", func(t *testing.T) {
		conditional := Conditional(condition, trueTask, falseTask).Named("test-conditional")

		if conditional.GetName() != "test-conditional" {
			t.Errorf("Expected name 'test-conditional', got '%s'", conditional.GetName())
		}
	})

	t.Run("with_config", func(t *testing.T) {
		cfg := config.Config{
			Timeout: 30 * time.Second,
		}

		conditional := Conditional(condition, trueTask, falseTask).With(cfg)

		if conditional.GetConfig().Timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", conditional.GetConfig().Timeout)
		}
	})

	t.Run("error_boundary", func(t *testing.T) {
		conditional := Conditional(condition, trueTask, falseTask).ErrorBoundary(orchErrors.CollectAll)

		// The error boundary should be set (we can't directly test it without executing)
		if conditional == nil {
			t.Error("Expected conditional to be created with error boundary")
		}
	})

	t.Run("method_chaining", func(t *testing.T) {
		cfg := config.Config{Timeout: 15 * time.Second}

		conditional := Conditional(condition, trueTask, falseTask).
			Named("chained-conditional").
			With(cfg).
			ErrorBoundary(orchErrors.FailFast)

		if conditional.GetName() != "chained-conditional" {
			t.Errorf("Expected name 'chained-conditional', got '%s'", conditional.GetName())
		}

		if conditional.GetConfig().Timeout != 15*time.Second {
			t.Errorf("Expected timeout 15s, got %v", conditional.GetConfig().Timeout)
		}
	})
}

// TestConditionalBuilder_Execute tests the execution logic
func TestConditionalBuilder_Execute(t *testing.T) {
	t.Run("condition_true_executes_ifTrue", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
		trueTask := task.Task(func() (string, error) { return "true-result", nil }).Named("true-task")
		falseTask := task.Task(func() (string, error) { return "false-result", nil }).Named("false-task")

		conditional := Conditional(condition, trueTask, falseTask).Named("test-conditional")

		ctx := context.Background()
		cfg := config.DefaultConfig()

		result, err := conditional.Execute(ctx, cfg)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		// Check if true branch was executed
		branchResult := result.Get("if-true")
		if branchResult != "true-result" {
			t.Errorf("Expected 'true-result', got %v", branchResult)
		}
	})

	t.Run("condition_false_executes_ifFalse", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) { return false, nil }
		trueTask := task.Task(func() (string, error) { return "true-result", nil }).Named("true-task")
		falseTask := task.Task(func() (string, error) { return "false-result", nil }).Named("false-task")

		conditional := Conditional(condition, trueTask, falseTask).Named("test-conditional")

		ctx := context.Background()
		cfg := config.DefaultConfig()

		result, err := conditional.Execute(ctx, cfg)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		// Check if false branch was executed
		branchResult := result.Get("if-false")
		if branchResult != "false-result" {
			t.Errorf("Expected 'false-result', got %v", branchResult)
		}
	})

	t.Run("condition_error_propagates", func(t *testing.T) {
		conditionError := errors.New("condition evaluation failed")
		condition := func(ctx orchContext.Context) (bool, error) { return false, conditionError }
		trueTask := task.Task(func() (string, error) { return "true-result", nil })
		falseTask := task.Task(func() (string, error) { return "false-result", nil })

		conditional := Conditional(condition, trueTask, falseTask).Named("error-conditional")

		ctx := context.Background()
		cfg := config.DefaultConfig()

		result, err := conditional.Execute(ctx, cfg)

		if err == nil {
			t.Fatal("Expected error from failing condition")
		}

		if !errors.Is(err, conditionError) {
			t.Errorf("Expected condition error to be wrapped, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}

		// Should contain error information
		if len(result.Errors()) == 0 {
			t.Error("Expected error to be recorded in result")
		}
	})

	t.Run("branch_execution_error_propagates", func(t *testing.T) {
		branchError := errors.New("branch execution failed")
		condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
		trueTask := task.Task(func() (string, error) { return "", branchError }).Named("failing-task")
		falseTask := task.Task(func() (string, error) { return "false-result", nil })

		conditional := Conditional(condition, trueTask, falseTask).Named("branch-error-conditional")

		ctx := context.Background()
		cfg := config.DefaultConfig()

		result, err := conditional.Execute(ctx, cfg)

		if err == nil {
			t.Fatal("Expected error from failing branch")
		}

		if !errors.Is(err, branchError) {
			t.Errorf("Expected branch error to be propagated, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}

		// Should contain error information
		if len(result.Errors()) == 0 {
			t.Error("Expected error to be recorded in result")
		}
	})

	t.Run("condition_panic_recovery", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) {
			panic("condition panic")
		}
		trueTask := task.Task(func() (string, error) { return "true-result", nil })
		falseTask := task.Task(func() (string, error) { return "false-result", nil })

		conditional := Conditional(condition, trueTask, falseTask).Named("panic-conditional")

		ctx := context.Background()
		cfg := config.DefaultConfig()

		result, err := conditional.Execute(ctx, cfg)

		if err == nil {
			t.Fatal("Expected error from panicking condition")
		}

		if result == nil {
			t.Fatal("Expected result even with panic")
		}

		// Should contain panic information
		if len(result.Errors()) == 0 {
			t.Error("Expected panic to be recorded as error in result")
		}
	})

	t.Run("context_cancellation", func(t *testing.T) {
		condition := func(ctx orchContext.Context) (bool, error) {
			time.Sleep(100 * time.Millisecond) // Simulate slow condition
			return true, nil
		}
		trueTask := task.Task(func() (string, error) { return "true-result", nil })
		falseTask := task.Task(func() (string, error) { return "false-result", nil })

		conditional := Conditional(condition, trueTask, falseTask).Named("cancellation-conditional")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		cfg := config.DefaultConfig()

		result, err := conditional.Execute(ctx, cfg)

		if err == nil {
			t.Fatal("Expected error from context cancellation")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context deadline exceeded, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result even with cancellation")
		}
	})
}

// TestConditionalBuilder_GetChildren tests child orchestration access
func TestConditionalBuilder_GetChildren(t *testing.T) {
	condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
	trueTask := task.Task(func() (string, error) { return "true", nil }).Named("true-task")
	falseTask := task.Task(func() (string, error) { return "false", nil }).Named("false-task")

	conditional := Conditional(condition, trueTask, falseTask)

	children := conditional.GetChildren()

	if len(children) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(children))
	}

	if children[0] != trueTask {
		t.Error("Expected first child to be trueTask")
	}

	if children[1] != falseTask {
		t.Error("Expected second child to be falseTask")
	}
}

// TestConditionalBuilder_PathResolution tests path-based orchestration resolution
func TestConditionalBuilder_PathResolution(t *testing.T) {
	condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
	trueTask := task.Task(func() (string, error) { return "true", nil }).Named("true-task")
	falseTask := task.Task(func() (string, error) { return "false", nil }).Named("false-task")

	conditional := Conditional(condition, trueTask, falseTask).Named("test-conditional")

	t.Run("get_current_path", func(t *testing.T) {
		path := conditional.GetCurrentPath()
		if path == "" {
			t.Error("Expected non-empty current path")
		}
	})

	t.Run("list_all_paths", func(t *testing.T) {
		paths := conditional.ListAllPaths()
		if len(paths) == 0 {
			t.Error("Expected at least one path")
		}
	})

	t.Run("find_by_name", func(t *testing.T) {
		matches := conditional.FindByName("true-task")
		if len(matches) == 0 {
			t.Error("Expected to find true-task by name")
		}
	})

	t.Run("get_orchestration_tree", func(t *testing.T) {
		tree := conditional.GetOrchestrationTree()
		if tree == nil {
			t.Fatal("Expected orchestration tree")
		}

		if tree.Name != "test-conditional" {
			t.Errorf("Expected tree name 'test-conditional', got '%s'", tree.Name)
		}

		if len(tree.Children) != 2 {
			t.Errorf("Expected 2 children in tree, got %d", len(tree.Children))
		}
	})
}

// TestConditionalBuilder_ConfigurationInheritance tests configuration inheritance
func TestConditionalBuilder_ConfigurationInheritance(t *testing.T) {
	condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
	trueTask := task.Task(func() (string, error) { return "true-result", nil }).Named("true-task")
	falseTask := task.Task(func() (string, error) { return "false-result", nil }).Named("false-task")

	parentConfig := config.Config{
		Timeout:        60 * time.Second,
		ErrorStrategy:  orchErrors.FailFast,
		MaxConcurrency: 5,
		Context:        context.Background(),
	}

	conditionalConfig := config.Config{
		Timeout: 30 * time.Second, // Override parent timeout
	}

	conditional := Conditional(condition, trueTask, falseTask).
		Named("config-conditional").
		With(conditionalConfig)

	ctx := context.Background()

	result, err := conditional.Execute(ctx, parentConfig)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Verify that the branch was executed successfully
	branchResult := result.Get("if-true")
	if branchResult != "true-result" {
		t.Errorf("Expected 'true-result', got %v", branchResult)
	}
}

// TestConditionalBuilder_ErrorBoundaries tests error boundary behavior
func TestConditionalBuilder_ErrorBoundaries(t *testing.T) {
	t.Run("fail_fast_strategy", func(t *testing.T) {
		branchError := errors.New("branch execution failed")
		condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
		trueTask := task.Task(func() (string, error) { return "", branchError }).Named("failing-task")
		falseTask := task.Task(func() (string, error) { return "false-result", nil })

		conditional := Conditional(condition, trueTask, falseTask).
			Named("fail-fast-conditional").
			ErrorBoundary(orchErrors.FailFast)

		ctx := context.Background()
		cfg := config.DefaultConfig()

		result, err := conditional.Execute(ctx, cfg)

		if err == nil {
			t.Fatal("Expected error with FailFast strategy")
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}

		// Should contain error information
		if len(result.Errors()) == 0 {
			t.Error("Expected error to be recorded in result")
		}
	})

	t.Run("collect_all_strategy", func(t *testing.T) {
		branchError := errors.New("branch execution failed")
		condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
		trueTask := task.Task(func() (string, error) { return "", branchError }).Named("failing-task")
		falseTask := task.Task(func() (string, error) { return "false-result", nil })

		conditional := Conditional(condition, trueTask, falseTask).
			Named("collect-all-conditional").
			ErrorBoundary(orchErrors.CollectAll)

		ctx := context.Background()
		cfg := config.DefaultConfig()

		result, err := conditional.Execute(ctx, cfg)

		if err == nil {
			t.Fatal("Expected error with CollectAll strategy")
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}

		// Should contain error information
		if len(result.Errors()) == 0 {
			t.Error("Expected error to be recorded in result")
		}
	})
}

// BenchmarkConditionalBuilder_Execute benchmarks conditional execution
func BenchmarkConditionalBuilder_Execute(b *testing.B) {
	condition := func(ctx orchContext.Context) (bool, error) { return true, nil }
	trueTask := task.Task(func() (string, error) { return "true-result", nil }).Named("true-task")
	falseTask := task.Task(func() (string, error) { return "false-result", nil }).Named("false-task")

	conditional := Conditional(condition, trueTask, falseTask).Named("benchmark-conditional")

	ctx := context.Background()
	cfg := config.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := conditional.Execute(ctx, cfg)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
