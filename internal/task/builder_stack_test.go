package task

import "testing"

func TestTaskBuilderCaptureStack(t *testing.T) {
	tk := Task(func() (string, error) { return "test", nil })
	stk := tk.captureStack()
	if len(stk) == 0 {
		t.Error("expected non-empty stack")
	}
	if !contains(string(stk), "TestTaskBuilderCaptureStack") {
		t.Error("stack should contain function name")
	}
}
