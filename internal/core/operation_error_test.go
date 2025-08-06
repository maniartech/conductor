package core

import (
	"errors"
	"testing"
	"time"
)

func TestOperationError(t *testing.T) {
	originalErr := errors.New("original error")
	opErr := OperationError{
		Error:     originalErr,
		Index:     5,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
		OpID:      "test-operation",
		Stack:     []byte("stack trace here"),
	}

	// Test Error() method
	if opErr.Error.Error() != "original error" {
		t.Errorf("Expected 'original error', got %q", opErr.Error.Error())
	}

	// Test field access
	if opErr.Index != 5 {
		t.Errorf("Expected Index 5, got %d", opErr.Index)
	}

	if opErr.Duration != 100*time.Millisecond {
		t.Errorf("Expected Duration 100ms, got %v", opErr.Duration)
	}

	if opErr.OpID != "test-operation" {
		t.Errorf("Expected OpID 'test-operation', got %q", opErr.OpID)
	}

	if string(opErr.Stack) != "stack trace here" {
		t.Errorf("Expected stack trace, got %q", string(opErr.Stack))
	}
}

func TestOperationErrorWithNilError(t *testing.T) {
	opErr := OperationError{
		Error:     nil,
		Index:     0,
		Duration:  0,
		Timestamp: time.Now(),
		OpID:      "nil-error-test",
		Stack:     nil,
	}

	// Should handle nil error gracefully
	if opErr.Error != nil {
		t.Error("Error should be nil")
	}

	// Other fields should still be accessible
	if opErr.OpID != "nil-error-test" {
		t.Error("OpID should be accessible even with nil error")
	}
}

func TestOperationErrorTimestamp(t *testing.T) {
	start := time.Now()
	opErr := OperationError{
		Error:     errors.New("test"),
		Timestamp: time.Now(),
	}
	end := time.Now()

	// Timestamp should be within the test execution window
	if opErr.Timestamp.Before(start) || opErr.Timestamp.After(end) {
		t.Error("Timestamp should be within execution window")
	}
}

func TestOperationErrorDuration(t *testing.T) {
	opErr := OperationError{
		Error:    errors.New("test"),
		Duration: 250 * time.Millisecond,
	}

	if opErr.Duration != 250*time.Millisecond {
		t.Errorf("Expected Duration 250ms, got %v", opErr.Duration)
	}

	// Test zero duration
	opErr.Duration = 0
	if opErr.Duration != 0 {
		t.Error("Duration should be 0")
	}

	// Test negative duration (edge case)
	opErr.Duration = -100 * time.Millisecond
	if opErr.Duration != -100*time.Millisecond {
		t.Error("Should handle negative duration")
	}
}

func TestOperationErrorStack(t *testing.T) {
	stackTrace := []byte("goroutine 1 [running]:\nmain.main()\n\t/path/to/main.go:10 +0x20")
	opErr := OperationError{
		Error: errors.New("test"),
		Stack: stackTrace,
	}

	if string(opErr.Stack) != string(stackTrace) {
		t.Error("Stack trace should be preserved")
	}

	// Test empty stack
	opErr.Stack = []byte{}
	if len(opErr.Stack) != 0 {
		t.Error("Empty stack should be preserved")
	}

	// Test nil stack
	opErr.Stack = nil
	if opErr.Stack != nil {
		t.Error("Nil stack should be preserved")
	}
}

func TestOperationErrorIndex(t *testing.T) {
	tests := []struct {
		index    int
		expected int
	}{
		{0, 0},
		{1, 1},
		{100, 100},
		{-1, -1}, // Edge case
	}

	for _, test := range tests {
		opErr := OperationError{
			Error: errors.New("test"),
			Index: test.index,
		}

		if opErr.Index != test.expected {
			t.Errorf("Expected Index %d, got %d", test.expected, opErr.Index)
		}
	}
}

func TestOperationErrorOpID(t *testing.T) {
	tests := []string{
		"simple-id",
		"complex-operation-id-with-dashes",
		"operation_with_underscores",
		"Operation123",
		"", // Empty string
	}

	for _, opID := range tests {
		opErr := OperationError{
			Error: errors.New("test"),
			OpID:  opID,
		}

		if opErr.OpID != opID {
			t.Errorf("Expected OpID %q, got %q", opID, opErr.OpID)
		}
	}
}

func TestOperationErrorComparison(t *testing.T) {
	now := time.Now()
	stack := []byte("stack trace")

	opErr1 := OperationError{
		Error:     errors.New("same error"),
		Index:     1,
		Duration:  100 * time.Millisecond,
		Timestamp: now,
		OpID:      "op-1",
		Stack:     stack,
	}

	opErr2 := OperationError{
		Error:     errors.New("same error"),
		Index:     1,
		Duration:  100 * time.Millisecond,
		Timestamp: now,
		OpID:      "op-1",
		Stack:     stack,
	}

	// Note: These are different instances, so they won't be equal
	// This test just verifies that fields can be compared individually
	if opErr1.Index != opErr2.Index {
		t.Error("Indexes should be equal")
	}

	if opErr1.Duration != opErr2.Duration {
		t.Error("Durations should be equal")
	}

	if opErr1.OpID != opErr2.OpID {
		t.Error("OpIDs should be equal")
	}

	if !opErr1.Timestamp.Equal(opErr2.Timestamp) {
		t.Error("Timestamps should be equal")
	}

	if string(opErr1.Stack) != string(opErr2.Stack) {
		t.Error("Stack traces should be equal")
	}
}

func TestOperationErrorWithComplexError(t *testing.T) {
	// Test with wrapped error
	innerErr := errors.New("inner error")
	wrappedErr := errors.New("wrapped: " + innerErr.Error())

	opErr := OperationError{
		Error: wrappedErr,
		OpID:  "wrapped-error-test",
	}

	if opErr.Error.Error() != "wrapped: inner error" {
		t.Errorf("Expected wrapped error message, got %q", opErr.Error.Error())
	}
}

// Benchmark tests
func BenchmarkOperationErrorCreation(b *testing.B) {
	err := errors.New("benchmark error")
	stack := []byte("benchmark stack trace")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = OperationError{
			Error:     err,
			Index:     i,
			Duration:  time.Millisecond,
			Timestamp: time.Now(),
			OpID:      "benchmark-op",
			Stack:     stack,
		}
	}
}

func BenchmarkOperationErrorAccess(b *testing.B) {
	opErr := OperationError{
		Error:     errors.New("benchmark error"),
		Index:     42,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
		OpID:      "benchmark-op",
		Stack:     []byte("benchmark stack"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = opErr.Error.Error()
		_ = opErr.Index
		_ = opErr.Duration
		_ = opErr.Timestamp
		_ = opErr.OpID
		_ = opErr.Stack
	}
}
