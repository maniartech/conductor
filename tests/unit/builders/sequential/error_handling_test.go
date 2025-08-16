package sequential

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	errorspkg "github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/pkg/builders/task"

	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
)

// TestEnhancedErrorHandling_FailFast tests enhanced fail-fast error handling
func TestEnhancedErrorHandling_FailFast(t *testing.T) {
	// Create a sequential with an error in the middle
	seq := Sequential(
		task.Task(func() (string, error) { return "step1", nil }).Named("step1"),
		task.Task(func() (int, error) { return 0, errors.New("step2 failed") }).Named("step2"),
		task.Task(func() (bool, error) { return true, nil }).Named("step3"),
	).Named("enhanced-fail-fast-test")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.FailFast}

	result, err := seq.Execute(ctx, cfg)

	// Should get an enhanced error with detailed report
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Error message should contain detailed report
	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "Detailed Report:") {
		t.Error("Expected enhanced error with detailed report")
	}
	if !strings.Contains(errorMsg, "Orchestration Name: enhanced-fail-fast-test") {
		t.Error("Expected orchestration name in error report")
	}
	if !strings.Contains(errorMsg, "Error Strategy: FailFast") {
		t.Error("Expected error strategy in error report")
	}

	// Should have partial results (step1 succeeded)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Get("step1") != "step1" {
		t.Error("Expected step1 result")
	}
	if result.Get("step3") != nil {
		t.Error("Expected no step3 result with FailFast")
	}

	// Should have one error in result
	if len(result.Errors()) != 1 {
		t.Errorf("Expected 1 error in result, got %d", len(result.Errors()))
	}

	t.Logf("Enhanced fail-fast error handling test passed")
}

// TestEnhancedErrorHandling_CollectAll tests enhanced collect-all error handling
func TestEnhancedErrorHandling_CollectAll(t *testing.T) {
	// Create a sequential with multiple errors
	seq := Sequential(
		task.Task(func() (string, error) { return "step1", nil }).Named("step1"),
		task.Task(func() (int, error) { return 0, errors.New("step2 failed") }).Named("step2"),
		task.Task(func() (bool, error) { return true, nil }).Named("step3"),
		task.Task(func() (string, error) { return "", errors.New("step4 failed") }).Named("step4"),
	).Named("enhanced-collect-all-test")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.CollectAll}

	result, err := seq.Execute(ctx, cfg)

	// Should get an enhanced error with detailed report
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Error message should contain detailed report
	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "Detailed Report:") {
		t.Error("Expected enhanced error with detailed report")
	}
	if !strings.Contains(errorMsg, "Orchestration Name: enhanced-collect-all-test") {
		t.Error("Expected orchestration name in error report")
	}
	if !strings.Contains(errorMsg, "Error Strategy: CollectAll") {
		t.Error("Expected error strategy in error report")
	}
	if !strings.Contains(errorMsg, "Total Errors: 2") {
		t.Error("Expected total error count in report")
	}

	// Should have results from successful steps
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Get("step1") != "step1" {
		t.Error("Expected step1 result")
	}
	if result.Get("step3") != true {
		t.Error("Expected step3 result")
	}

	// Should have two errors in result
	if len(result.Errors()) != 2 {
		t.Errorf("Expected 2 errors in result, got %d", len(result.Errors()))
	}

	t.Logf("Enhanced collect-all error handling test passed")
}

// TestErrorBoundaryHandler tests the error boundary handler functionality
func TestErrorBoundaryHandler(t *testing.T) {
	// Test FailFast strategy
	handler := errorspkg.NewErrorBoundaryHandler(errorspkg.FailFast, "test-boundary", nil)

	// First error should stop execution
	shouldContinue := handler.HandleError(errors.New("first error"), 0, "step1", time.Millisecond, nil)
	if shouldContinue {
		t.Error("Expected FailFast to stop execution on first error")
	}

	if !handler.HasErrors() {
		t.Error("Expected handler to have errors")
	}

	if handler.GetErrorCount() != 1 {
		t.Errorf("Expected 1 error, got %d", handler.GetErrorCount())
	}

	finalError := handler.GetFinalError()
	if finalError == nil {
		t.Error("Expected final error")
	}
	if !strings.Contains(finalError.Error(), "test-boundary") {
		t.Error("Expected boundary name in final error")
	}

	// Test CollectAll strategy
	collectHandler := errorspkg.NewErrorBoundaryHandler(errorspkg.CollectAll, "collect-boundary", nil)

	// First error should continue execution
	shouldContinue = collectHandler.HandleError(errors.New("first error"), 0, "step1", time.Millisecond, nil)
	if !shouldContinue {
		t.Error("Expected CollectAll to continue execution on first error")
	}

	// Second error should also continue
	shouldContinue = collectHandler.HandleError(errors.New("second error"), 1, "step2", time.Millisecond, nil)
	if !shouldContinue {
		t.Error("Expected CollectAll to continue execution on second error")
	}

	if collectHandler.GetErrorCount() != 2 {
		t.Errorf("Expected 2 errors, got %d", collectHandler.GetErrorCount())
	}

	allErrors := collectHandler.GetAllErrors()
	if len(allErrors) != 2 {
		t.Errorf("Expected 2 errors in GetAllErrors, got %d", len(allErrors))
	}

	t.Logf("Error boundary handler test passed")
}

// TestErrorContextCreation tests error context creation and updates
func TestErrorContextCreation(t *testing.T) {
	orchestrations := []orchestration.Orchestration{
		task.Task(func() (string, error) { return "test", nil }),
		task.Task(func() (int, error) { return 42, nil }),
	}

	ctx := errorspkg.CreateErrorContext("test-seq", "test-id", "sequential", len(orchestrations), errorspkg.FailFast, nil)

	if ctx.OrchestrationName != "test-seq" {
		t.Errorf("Expected orchestration name 'test-seq', got %s", ctx.OrchestrationName)
	}
	if ctx.OrchestrationID != "test-id" {
		t.Errorf("Expected orchestration ID 'test-id', got %s", ctx.OrchestrationID)
	}
	if ctx.TotalSteps != 2 {
		t.Errorf("Expected 2 total steps, got %d", ctx.TotalSteps)
	}
	if ctx.ErrorStrategy != errorspkg.FailFast {
		t.Errorf("Expected FailFast strategy, got %v", ctx.ErrorStrategy)
	}

	// Test context updates
	errorspkg.UpdateErrorContext(ctx, 1, time.Second, false)
	if ctx.CompletedSteps != 1 {
		t.Errorf("Expected 1 completed step, got %d", ctx.CompletedSteps)
	}
	if ctx.ExecutionTime != time.Second {
		t.Errorf("Expected 1 second execution time, got %v", ctx.ExecutionTime)
	}

	// Test failed step setting
	errorspkg.SetFailedStep(ctx, 1, "failed-step", []byte("stack trace"))
	if ctx.FailedStep != 1 {
		t.Errorf("Expected failed step 1, got %d", ctx.FailedStep)
	}
	if ctx.FailedStepName != "failed-step" {
		t.Errorf("Expected failed step name 'failed-step', got %s", ctx.FailedStepName)
	}

	t.Logf("Error context creation test passed")
}

// TestEnhancedErrorReporting tests the enhanced error reporting functionality
func TestEnhancedErrorReporting(t *testing.T) {
	// Create error context and handler
	orchestrations := []orchestration.Orchestration{
		task.Task(func() (string, error) { return "test", nil }),
	}

	ctx := errorspkg.CreateErrorContext("test-reporting", "report-id", "sequential", len(orchestrations), errorspkg.CollectAll, nil)
	handler := errorspkg.NewErrorBoundaryHandler(errorspkg.CollectAll, "report-boundary", nil)

	// Add some errors
	handler.HandleError(errors.New("test error 1"), 0, "step1", time.Millisecond, ctx)
	handler.HandleError(errors.New("test error 2"), 1, "step2", time.Millisecond*2, ctx)

	// Create reporter and generate report
	reporter := errorspkg.NewEnhancedErrorReporting(ctx, handler)
	report := reporter.GenerateErrorReport()

	// Verify report contains expected information
	if !strings.Contains(report, "Orchestration Name: test-reporting") {
		t.Error("Expected orchestration name in report")
	}
	if !strings.Contains(report, "Orchestration ID: report-id") {
		t.Error("Expected orchestration ID in report")
	}
	if !strings.Contains(report, "Error Boundary: report-boundary") {
		t.Error("Expected error boundary in report")
	}
	if !strings.Contains(report, "Total Errors: 2") {
		t.Error("Expected total error count in report")
	}
	if !strings.Contains(report, "test error 1") {
		t.Error("Expected first error in report")
	}
	if !strings.Contains(report, "test error 2") {
		t.Error("Expected second error in report")
	}

	t.Logf("Enhanced error reporting test passed")
}

// TestPanicRecovery tests panic recovery within error boundaries
func TestPanicRecovery(t *testing.T) {
	// Create a sequential that panics
	seq := Sequential(
		task.Task(func() (string, error) { return "step1", nil }).Named("step1"),
		task.Task(func() (int, error) { panic("test panic") }).Named("step2"),
		task.Task(func() (bool, error) { return true, nil }).Named("step3"),
	).Named("panic-test")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.FailFast}

	result, err := seq.Execute(ctx, cfg)

	// Should recover from panic and return error
	if err == nil {
		t.Fatal("Expected error from panic recovery, got nil")
	}

	// Error should contain panic information
	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "panic recovered") {
		t.Error("Expected panic recovery information in error")
	}

	// Should have result from step1
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Get("step1") != "step1" {
		t.Error("Expected step1 result")
	}

	// Should have error in result
	if len(result.Errors()) == 0 {
		t.Error("Expected error in result from panic")
	}

	t.Logf("Panic recovery test passed")
}

// BenchmarkEnhancedErrorHandling benchmarks the enhanced error handling performance
func BenchmarkEnhancedErrorHandling(b *testing.B) {
	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errorspkg.FailFast}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create new sequential for each iteration
		testSeq := Sequential(
			task.Task(func() (int, error) { return 1, nil }),
			task.Task(func() (int, error) { return 0, errors.New("test error") }),
			task.Task(func() (int, error) { return 3, nil }),
		).Named("benchmark-test")

		result, err := testSeq.Execute(ctx, cfg)
		if err == nil {
			b.Fatal("Expected error")
		}
		if result == nil {
			b.Fatal("Expected non-nil result")
		}
	}
}
