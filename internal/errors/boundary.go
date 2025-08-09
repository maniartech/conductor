// Package errors provides error boundary and context management for all orchestration types.
// This file implements common error handling components that can be used across
// sequential, concurrent, and other orchestration types.
package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorContext provides rich context information for orchestration errors.
// It captures detailed information about the execution environment when errors occur,
// enabling comprehensive debugging and observability across all orchestration types.
type ErrorContext struct {
	OrchestrationName string         // Name of the orchestration
	OrchestrationID   string         // Unique ID of the orchestration
	OrchestrationKind string         // Type of orchestration (sequential, concurrent, etc.)
	TotalSteps        int            // Total number of steps in the orchestration
	CompletedSteps    int            // Number of steps completed before error
	FailedStep        int            // Index of the step that failed
	FailedStepName    string         // Name of the failed step
	ExecutionTime     time.Duration  // Total execution time before failure
	ErrorStrategy     ErrorStrategy  // Error strategy being used
	ErrorBoundary     *ErrorStrategy // Error boundary if set
	ContextCancelled  bool           // Whether context was cancelled
	StackTrace        []byte         // Stack trace at error point
	Metadata          map[string]any // Additional orchestration-specific metadata
}

// ErrorBoundaryHandler manages error boundary containment within orchestration blocks.
// It provides sophisticated error isolation and propagation control, ensuring that
// errors are contained within appropriate scopes and don't leak beyond error boundaries.
type ErrorBoundaryHandler struct {
	strategy      ErrorStrategy
	boundaryName  string
	parentContext *ErrorContext
	errorCount    int
	firstError    error
	allErrors     []OperationError
}

// NewErrorBoundaryHandler creates a new error boundary handler for any orchestration type.
// It sets up error containment with the specified strategy and provides rich error context.
//
// Parameters:
//   - strategy: The error handling strategy (FailFast or CollectAll)
//   - boundaryName: Name of the error boundary for identification
//   - parentContext: Parent error context for nested orchestrations
//
// Returns:
//   - *ErrorBoundaryHandler: A new error boundary handler instance
//
// Example:
//
//	handler := NewErrorBoundaryHandler(errors.CollectAll, "user-processing", nil)
//	defer handler.HandlePanic() // Ensure panics are contained within boundary
func NewErrorBoundaryHandler(strategy ErrorStrategy, boundaryName string, parentContext *ErrorContext) *ErrorBoundaryHandler {
	return &ErrorBoundaryHandler{
		strategy:      strategy,
		boundaryName:  boundaryName,
		parentContext: parentContext,
		allErrors:     make([]OperationError, 0),
	}
}

// HandleError processes an error within the error boundary.
// It applies the configured error strategy and collects rich error metadata.
//
// Parameters:
//   - err: The error that occurred
//   - stepIndex: Index of the step where the error occurred
//   - stepName: Name of the step where the error occurred
//   - duration: Duration of the failed step execution
//   - context: Rich error context information
//
// Returns:
//   - bool: Whether execution should continue (true) or stop (false)
//
// Example:
//
//	shouldContinue := handler.HandleError(err, 2, "validate-user", duration, errorCtx)
//	if !shouldContinue {
//	    return handler.GetFinalError()
//	}
func (ebh *ErrorBoundaryHandler) HandleError(err error, stepIndex int, stepName string, duration time.Duration, context *ErrorContext) bool {
	if err == nil {
		return true
	}

	ebh.errorCount++
	if ebh.firstError == nil {
		ebh.firstError = err
	}

	// Create rich operation error with full context
	opError := OperationError{
		Error:     err,
		Index:     stepIndex,
		Duration:  duration,
		Timestamp: time.Now(),
		OpID:      fmt.Sprintf("%s.%s", ebh.boundaryName, stepName),
		Stack:     ebh.CaptureStackTrace(),
	}

	ebh.allErrors = append(ebh.allErrors, opError)

	// Apply error strategy
	switch ebh.strategy {
	case FailFast:
		return false // Stop execution immediately
	case CollectAll:
		return true // Continue execution to collect all errors
	default:
		return false // Default to fail-fast for unknown strategies
	}
}

// HandlePanic recovers from panics within the error boundary and converts them to errors.
// This ensures that panics don't escape the error boundary and are properly handled.
//
// Example:
//
//	defer func() {
//	    if panicErr := handler.HandlePanic(); panicErr != nil {
//	        log.Printf("Panic recovered within error boundary: %v", panicErr)
//	    }
//	}()
func (ebh *ErrorBoundaryHandler) HandlePanic() error {
	if r := recover(); r != nil {
		panicErr := fmt.Errorf("panic recovered in error boundary '%s': %v", ebh.boundaryName, r)

		// Create panic error with stack trace
		opError := OperationError{
			Error:     panicErr,
			Index:     -1, // Special index for panic errors
			Duration:  0,
			Timestamp: time.Now(),
			OpID:      fmt.Sprintf("%s.panic", ebh.boundaryName),
			Stack:     ebh.CaptureStackTrace(),
		}

		ebh.allErrors = append(ebh.allErrors, opError)
		ebh.errorCount++
		if ebh.firstError == nil {
			ebh.firstError = panicErr
		}

		return panicErr
	}
	return nil
}

// GetFinalError returns the final error based on the error strategy and collected errors.
// For FailFast, returns the first error. For CollectAll, returns an aggregated error.
//
// Returns:
//   - error: The final error to return from the orchestration
//
// Example:
//
//	if handler.HasErrors() {
//	    return result, handler.GetFinalError()
//	}
func (ebh *ErrorBoundaryHandler) GetFinalError() error {
	if ebh.errorCount == 0 {
		return nil
	}

	switch ebh.strategy {
	case FailFast:
		return fmt.Errorf("orchestration failed in error boundary '%s': %w", ebh.boundaryName, ebh.firstError)
	case CollectAll:
		return fmt.Errorf("orchestration completed with %d errors in error boundary '%s', first error: %w",
			ebh.errorCount, ebh.boundaryName, ebh.firstError)
	default:
		return ebh.firstError
	}
}

// HasErrors returns true if any errors were collected within the error boundary.
//
// Returns:
//   - bool: Whether any errors occurred within the boundary
func (ebh *ErrorBoundaryHandler) HasErrors() bool {
	return ebh.errorCount > 0
}

// GetAllErrors returns all errors collected within the error boundary.
// This provides comprehensive error information for debugging and observability.
//
// Returns:
//   - []OperationError: All errors that occurred within the boundary
func (ebh *ErrorBoundaryHandler) GetAllErrors() []OperationError {
	// Return a copy to prevent external modification
	errorsCopy := make([]OperationError, len(ebh.allErrors))
	copy(errorsCopy, ebh.allErrors)
	return errorsCopy
}

// GetErrorCount returns the total number of errors collected within the error boundary.
//
// Returns:
//   - int: Total number of errors
func (ebh *ErrorBoundaryHandler) GetErrorCount() int {
	return ebh.errorCount
}

// CaptureStackTrace captures the current stack trace for error reporting.
// This provides detailed debugging information when errors occur.
func (ebh *ErrorBoundaryHandler) CaptureStackTrace() []byte {
	stack := make([]byte, 4096)
	stackSize := runtime.Stack(stack, false)
	return stack[:stackSize]
}

// CreateErrorContext creates a rich error context for any orchestration type.
// This context provides comprehensive information about the execution environment.
//
// Parameters:
//   - orchestrationName: Name of the orchestration
//   - orchestrationID: Unique ID of the orchestration
//   - orchestrationKind: Type of orchestration (sequential, concurrent, etc.)
//   - totalSteps: Total number of steps
//   - errorStrategy: Error strategy being used
//   - errorBoundary: Error boundary if set
//
// Returns:
//   - *ErrorContext: Rich error context information
//
// Example:
//
//	ctx := CreateErrorContext("user-pipeline", "uuid-123", "concurrent", 5, CollectAll, nil)
func CreateErrorContext(orchestrationName, orchestrationID, orchestrationKind string, totalSteps int,
	errorStrategy ErrorStrategy, errorBoundary *ErrorStrategy) *ErrorContext {

	return &ErrorContext{
		OrchestrationName: orchestrationName,
		OrchestrationID:   orchestrationID,
		OrchestrationKind: orchestrationKind,
		TotalSteps:        totalSteps,
		CompletedSteps:    0,
		FailedStep:        -1,
		FailedStepName:    "",
		ExecutionTime:     0,
		ErrorStrategy:     errorStrategy,
		ErrorBoundary:     errorBoundary,
		ContextCancelled:  false,
		StackTrace:        nil,
		Metadata:          make(map[string]any),
	}
}

// UpdateErrorContext updates the error context with current execution information.
// This should be called as execution progresses to maintain accurate context.
//
// Parameters:
//   - ctx: Error context to update
//   - completedSteps: Number of steps completed
//   - executionTime: Total execution time so far
//   - contextCancelled: Whether context was cancelled
func UpdateErrorContext(ctx *ErrorContext, completedSteps int, executionTime time.Duration, contextCancelled bool) {
	if ctx != nil {
		ctx.CompletedSteps = completedSteps
		ctx.ExecutionTime = executionTime
		ctx.ContextCancelled = contextCancelled
	}
}

// SetFailedStep updates the error context with information about the failed step.
// This should be called when a step fails to provide detailed error information.
//
// Parameters:
//   - ctx: Error context to update
//   - failedStep: Index of the failed step
//   - failedStepName: Name of the failed step
//   - stackTrace: Stack trace at the point of failure
func SetFailedStep(ctx *ErrorContext, failedStep int, failedStepName string, stackTrace []byte) {
	if ctx != nil {
		ctx.FailedStep = failedStep
		ctx.FailedStepName = failedStepName
		ctx.StackTrace = stackTrace
	}
}

// SetContextMetadata adds orchestration-specific metadata to the error context.
//
// Parameters:
//   - ctx: Error context to update
//   - key: Metadata key
//   - value: Metadata value
func SetContextMetadata(ctx *ErrorContext, key string, value any) {
	if ctx != nil && ctx.Metadata != nil {
		ctx.Metadata[key] = value
	}
}

// EnhancedErrorReporting provides detailed error reporting with rich context.
// This function generates comprehensive error reports for debugging and observability.
type EnhancedErrorReporting struct {
	context *ErrorContext
	handler *ErrorBoundaryHandler
}

// NewEnhancedErrorReporting creates a new enhanced error reporting instance.
//
// Parameters:
//   - context: Rich error context
//   - handler: Error boundary handler
//
// Returns:
//   - *EnhancedErrorReporting: New enhanced error reporting instance
func NewEnhancedErrorReporting(context *ErrorContext, handler *ErrorBoundaryHandler) *EnhancedErrorReporting {
	return &EnhancedErrorReporting{
		context: context,
		handler: handler,
	}
}

// GenerateErrorReport generates a comprehensive error report with all available context.
// This report includes execution details, error information, and debugging data.
//
// Returns:
//   - string: Comprehensive error report
func (eer *EnhancedErrorReporting) GenerateErrorReport() string {
	if eer.context == nil || eer.handler == nil {
		return "Error report unavailable: missing context or handler"
	}

	report := fmt.Sprintf(`
Orchestration Error Report
==========================
Orchestration Name: %s
Orchestration ID: %s
Orchestration Kind: %s
Error Boundary: %s
Error Strategy: %s
Total Steps: %d
Completed Steps: %d
Failed Step: %d (%s)
Execution Time: %v
Context Cancelled: %t
Total Errors: %d

Error Details:
`,
		eer.context.OrchestrationName,
		eer.context.OrchestrationID,
		eer.context.OrchestrationKind,
		eer.handler.boundaryName,
		eer.context.ErrorStrategy.String(),
		eer.context.TotalSteps,
		eer.context.CompletedSteps,
		eer.context.FailedStep,
		eer.context.FailedStepName,
		eer.context.ExecutionTime,
		eer.context.ContextCancelled,
		eer.handler.GetErrorCount(),
	)

	// Add individual error details
	for i, opError := range eer.handler.GetAllErrors() {
		report += fmt.Sprintf(`
Error %d:
  Operation ID: %s
  Index: %d
  Duration: %v
  Timestamp: %v
  Error: %v
`, i+1, opError.OpID, opError.Index, opError.Duration, opError.Timestamp, opError.Error)
	}

	// Add metadata if present
	if len(eer.context.Metadata) > 0 {
		report += "\nMetadata:\n"
		for key, value := range eer.context.Metadata {
			report += fmt.Sprintf("  %s: %v\n", key, value)
		}
	}

	return report
}

// LogErrorReport logs the comprehensive error report using the provided logging function.
// This enables integration with various logging frameworks.
//
// Parameters:
//   - logFunc: Function to use for logging (e.g., log.Printf)
func (eer *EnhancedErrorReporting) LogErrorReport(logFunc func(format string, args ...interface{})) {
	if logFunc != nil {
		logFunc("%s", eer.GenerateErrorReport())
	}
}

// CaptureCurrentStackTrace is a utility function to capture stack trace from any location.
// This is useful when you need to capture stack traces outside of the ErrorBoundaryHandler.
func CaptureCurrentStackTrace() []byte {
	stack := make([]byte, 4096)
	stackSize := runtime.Stack(stack, false)
	return stack[:stackSize]
}
