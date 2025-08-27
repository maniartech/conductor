package task

import (
	"context"
	systemErrors "errors"
	"fmt"
	"testing"

	"github.com/maniartech/orchestrator/pkg/config"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
)

func TestTaskBuilderCaptureStack(t *testing.T) {
	tk := Task(func(ctx orchContext.Context) (string, error) { return "test", nil })
	stk := tk.captureStack()
	if len(stk) == 0 {
		t.Error("expected non-empty stack")
	}
	if !contains(string(stk), "TestTaskBuilderCaptureStack") {
		t.Error("stack should contain function name")
	}
}

func TestTaskBuilderSafeExecute(t *testing.T) {
	cases := []struct {
		name        string
		fn          func(ctx orchContext.Context) (string, error)
		expectErr   bool
		expectPanic bool
	}{
		{"successful", func(ctx orchContext.Context) (string, error) { return "success", nil }, false, false},
		{"error", func(ctx orchContext.Context) (string, error) { return "", systemErrors.New("task error") }, true, false},
		{"panic", func(ctx orchContext.Context) (string, error) { panic("task panic") }, true, true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			res, err := Task(c.fn).safeExecute(context.Background(), config.Config{})
			if c.expectErr {
				if err == nil {
					t.Error("expected error")
				}
				if c.expectPanic && !contains(err.Error(), "panic recovered") {
					t.Error("expected panic recovery message")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res != "success" {
					t.Errorf("expected success got %v", res)
				}
			}
		})
	}
}

func ExampleTaskBuilder_safeExecute() {
	tk := Task(func(ctx orchContext.Context) (string, error) { return "Safe execution", nil })
	res, err := tk.safeExecute(context.Background(), config.Config{})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Result: %v\n", res)
	// Output:
	// Result: Safe execution
}
