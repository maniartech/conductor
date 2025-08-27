package orchestrator

import (
	"errors"
	"testing"

	"github.com/maniartech/orchestrator"
	. "github.com/maniartech/orchestrator"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
)

// TestConditional_NewSignature tests the new error-returning condition signature
func TestConditional_NewSignature(t *testing.T) {
	t.Run("condition_returns_true_with_no_error", func(t *testing.T) {
		// Create tasks
		trueTask := Task(func(ctx orchestrator.Context) (string, error) {
			return "true-branch", nil
		}).Named("true-task")

		falseTask := Task(func(ctx orchestrator.Context) (string, error) {
			return "false-branch", nil
		}).Named("false-task")

		// Create conditional with new signature
		conditional := Conditional(
			func(ctx orchContext.Context) (bool, error) {
				return true, nil
			},
			trueTask,
			falseTask,
		).Named("test-conditional")

		// Execute workflow
		workflow := Setup(conditional)
		result, err := workflow.Await()

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		// Should execute true branch
		branchResult := result.Get("if-true")
		if branchResult != "true-branch" {
			t.Errorf("Expected 'true-branch', got %v", branchResult)
		}
	})

	t.Run("condition_returns_false_with_no_error", func(t *testing.T) {
		// Create tasks
		trueTask := Task(func(ctx orchestrator.Context) (string, error) {
			return "true-branch", nil
		}).Named("true-task")

		falseTask := Task(func(ctx orchestrator.Context) (string, error) {
			return "false-branch", nil
		}).Named("false-task")

		// Create conditional with new signature
		conditional := Conditional(
			func(ctx orchContext.Context) (bool, error) {
				return false, nil
			},
			trueTask,
			falseTask,
		).Named("test-conditional")

		// Execute workflow
		workflow := Setup(conditional)
		result, err := workflow.Await()

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		// Should execute false branch
		branchResult := result.Get("if-false")
		if branchResult != "false-branch" {
			t.Errorf("Expected 'false-branch', got %v", branchResult)
		}
	})

	t.Run("condition_returns_error", func(t *testing.T) {
		conditionError := errors.New("condition evaluation failed")

		// Create tasks
		trueTask := Task(func(ctx orchestrator.Context) (string, error) {
			return "true-branch", nil
		}).Named("true-task")

		falseTask := Task(func(ctx orchestrator.Context) (string, error) {
			return "false-branch", nil
		}).Named("false-task")

		// Create conditional that returns an error
		conditional := Conditional(
			func(ctx orchContext.Context) (bool, error) {
				return false, conditionError
			},
			trueTask,
			falseTask,
		).Named("error-conditional")

		// Execute workflow
		workflow := Setup(conditional)
		result, err := workflow.Await()

		if err == nil {
			t.Fatal("Expected error from failing condition")
		}

		if !errors.Is(err, conditionError) {
			t.Errorf("Expected condition error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result even with error")
		}

		// Should contain error information
		if len(result.Errors()) == 0 {
			t.Error("Expected error to be recorded in result")
		}
	})

	t.Run("context_based_condition_with_error_handling", func(t *testing.T) {
		// Create tasks
		adminTask := Task(func(ctx orchestrator.Context) (string, error) {
			return "admin-access", nil
		}).Named("admin-task")

		userTask := Task(func(ctx orchestrator.Context) (string, error) {
			return "user-access", nil
		}).Named("user-task")

		// Create conditional with context-based condition that can return errors
		conditional := Conditional(
			func(ctx orchContext.Context) (bool, error) {
				role := ctx.Get("user_role")
				if role == nil {
					return false, errors.New("user_role not found in context")
				}
				roleStr, ok := role.(string)
				if !ok {
					return false, errors.New("user_role is not a string")
				}
				return roleStr == "admin", nil
			},
			adminTask,
			userTask,
		).Named("role-based-conditional")

		// Test with missing context value (should return error)
		workflow := Setup(conditional)
		result, err := workflow.Await()

		if err == nil {
			t.Fatal("Expected error when user_role is missing from context")
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

// TestConditional_PanicHandling tests that panics in conditions are properly handled
func TestConditional_PanicHandling(t *testing.T) {
	// Create tasks
	trueTask := Task(func(ctx orchestrator.Context) (string, error) {
		return "true-branch", nil
	}).Named("true-task")

	falseTask := Task(func(ctx orchestrator.Context) (string, error) {
		return "false-branch", nil
	}).Named("false-task")

	// Create conditional that panics
	conditional := Conditional(
		func(ctx orchContext.Context) (bool, error) {
			panic("condition panic")
		},
		trueTask,
		falseTask,
	).Named("panic-conditional")

	// Execute workflow
	workflow := Setup(conditional)
	result, err := workflow.Await()

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
}
