package sequential

import (
	"context"
	"errors"
	"testing"

	"github.com/maniartech/orchestrator/internal/config"
	errorspkg "github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/task"

	. "github.com/maniartech/orchestrator/internal/sequential"
)

func TestSequentialIntegration(t *testing.T) {
	// Create a simple sequential orchestration
	seq := Sequential(
		task.Task(func() (string, error) { return "step1", nil }).Named("step1"),
		task.Task(func() (int, error) { return 42, nil }).Named("step2"),
		task.Task(func() (bool, error) { return true, nil }).Named("step3"),
	).Named("integration-test")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.FailFast}

	result, err := seq.Execute(ctx, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Verify results
	if result.Get("step1") != "step1" {
		t.Errorf("Expected step1 result 'step1', got %v", result.Get("step1"))
	}

	if result.Get("step2") != 42 {
		t.Errorf("Expected step2 result 42, got %v", result.Get("step2"))
	}

	if result.Get("step3") != true {
		t.Errorf("Expected step3 result true, got %v", result.Get("step3"))
	}

	t.Logf("Sequential integration test passed successfully")
}

func TestSequentialWithError(t *testing.T) {
	// Create a sequential with an error
	seq := Sequential(
		task.Task(func() (string, error) { return "step1", nil }).Named("step1"),
		task.Task(func() (int, error) { return 0, errors.New("step2 failed") }).Named("step2"),
		task.Task(func() (bool, error) { return true, nil }).Named("step3"),
	).Named("error-test")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.FailFast}

	result, err := seq.Execute(ctx, cfg)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// With FailFast, should have step1 result but not step3
	if result.Get("step1") != "step1" {
		t.Errorf("Expected step1 result 'step1', got %v", result.Get("step1"))
	}

	if result.Get("step3") != nil {
		t.Errorf("Expected no step3 result with FailFast, got %v", result.Get("step3"))
	}

	// Should have one error
	if len(result.Errors()) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors()))
	}

	t.Logf("Sequential error test passed successfully")
}
