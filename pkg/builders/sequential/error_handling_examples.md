# Sequential Orchestration Error Handling Examples

This document provides comprehensive examples of the enhanced error handling capabilities in the sequential orchestration system.

## Overview

The sequential orchestration system provides military-grade error handling with:

- **Error Boundary Containment**: Errors are contained within defined scopes
- **Rich Error Context**: Detailed information about execution environment
- **Enhanced Error Reporting**: Comprehensive error reports for debugging
- **Panic Recovery**: Automatic panic recovery with detailed stack traces
- **Configurable Strategies**: FailFast and CollectAll error handling strategies

## Basic Error Handling

### FailFast Strategy

```go
// FailFast stops execution on the first error
seq := Sequential(
    Task(func() (string, error) { return "step1", nil }).Named("fetch-data"),
    Task(func() (int, error) { return 0, errors.New("validation failed") }).Named("validate-data"),
    Task(func() (bool, error) { return true, nil }).Named("process-data"),
).Named("data-pipeline")

ctx := context.Background()
config := Config{ErrorStrategy: FailFast}

result, err := seq.Execute(ctx, config)
if err != nil {
    // Error contains detailed report:
    // - Sequential name and ID
    // - Failed step information
    // - Execution timing
    // - Stack traces
    log.Printf("Pipeline failed: %v", err)
}
```

### CollectAll Strategy

```go
// CollectAll continues execution and collects all errors
seq := Sequential(
    Task(func() (string, error) { return "step1", nil }).Named("fetch-user"),
    Task(func() (int, error) { return 0, errors.New("email failed") }).Named("send-email"),
    Task(func() (bool, error) { return false, errors.New("sms failed") }).Named("send-sms"),
    Task(func() (string, error) { return "success", nil }).Named("log-activity"),
).Named("notification-pipeline")

ctx := context.Background()
config := Config{ErrorStrategy: CollectAll}

result, err := seq.Execute(ctx, config)
if err != nil {
    // Error contains comprehensive report of all failures
    log.Printf("Pipeline completed with errors: %v", err)
    
    // Access individual results
    if result.Get("fetch-user") != nil {
        log.Println("User data was fetched successfully")
    }
    if result.Get("log-activity") != nil {
        log.Println("Activity was logged successfully")
    }
    
    // Examine all errors
    for i, opErr := range result.Errors() {
        log.Printf("Error %d: %s failed after %v: %v", 
            i+1, opErr.OpID, opErr.Duration, opErr.Error)
    }
}
```

## Error Boundary Configuration

### Setting Error Boundaries

```go
// Override global error strategy with error boundary
seq := Sequential(
    Task(criticalOperation).Named("critical-step"),
    Task(optionalOperation).Named("optional-step"),
    Task(cleanupOperation).Named("cleanup-step"),
).Named("mixed-pipeline").
ErrorBoundary(CollectAll) // Override to continue despite errors

// Global config uses FailFast, but this sequential uses CollectAll
globalConfig := Config{ErrorStrategy: FailFast}
result, err := seq.Execute(ctx, globalConfig)
```

### Nested Error Boundaries

```go
// Nested sequential orchestrations with different error strategies
criticalSection := Sequential(
    Task(authenticateUser).Named("auth"),
    Task(validatePermissions).Named("permissions"),
).Named("critical-section").
ErrorBoundary(FailFast) // Must succeed

optionalSection := Sequential(
    Task(sendNotification).Named("notify"),
    Task(logAnalytics).Named("analytics"),
).Named("optional-section").
ErrorBoundary(CollectAll) // Continue despite failures

mainPipeline := Sequential(
    criticalSection,
    Task(processRequest).Named("process"),
    optionalSection,
).Named("main-pipeline")

result, err := mainPipeline.Execute(ctx, config)
```

## Advanced Error Context

### Rich Error Information

```go
seq := Sequential(
    Task(func() (string, error) { 
        time.Sleep(100 * time.Millisecond)
        return "", errors.New("database connection failed") 
    }).Named("connect-db"),
    Task(func() (int, error) { return 42, nil }).Named("process-data"),
).Named("database-pipeline")

result, err := seq.Execute(ctx, config)
if err != nil {
    // Error message includes detailed report:
    /*
    Sequential Orchestration Error Report
    =====================================
    Sequential Name: database-pipeline
    Sequential ID: sequential-database-pipeline
    Error Boundary: sequential-database-pipeline
    Error Strategy: FailFast
    Total Steps: 2
    Completed Steps: 0
    Failed Step: 0 (connect-db)
    Execution Time: 100ms
    Context Cancelled: false
    Total Errors: 1

    Error Details:
    Error 1:
      Operation ID: sequential-database-pipeline.connect-db
      Index: 0
      Duration: 100ms
      Timestamp: 2024-01-01T10:00:00Z
      Error: database connection failed
    */
}
```

### Custom Error Context

```go
// Access error details programmatically
result, err := seq.Execute(ctx, config)
if err != nil && result != nil {
    errors := result.Errors()
    for _, opErr := range errors {
        log.Printf("Operation %s (index %d) failed after %v: %v",
            opErr.OpID, opErr.Index, opErr.Duration, opErr.Error)
        
        // Stack trace is available for debugging
        if len(opErr.Stack) > 0 {
            log.Printf("Stack trace: %s", string(opErr.Stack))
        }
    }
}
```

## Panic Recovery

### Automatic Panic Recovery

```go
seq := Sequential(
    Task(func() (string, error) { return "step1", nil }).Named("safe-step"),
    Task(func() (int, error) { 
        panic("unexpected panic occurred") 
    }).Named("panic-step"),
    Task(func() (bool, error) { return true, nil }).Named("cleanup-step"),
).Named("panic-recovery-test")

result, err := seq.Execute(ctx, config)
// Panic is automatically recovered and converted to error
// Error report includes panic information and stack trace
if err != nil {
    log.Printf("Pipeline failed with panic: %v", err)
    // Contains: "panic recovered in error boundary 'sequential-panic-recovery-test': unexpected panic occurred"
}
```

## Context Cancellation

### Timeout Handling

```go
// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

seq := Sequential(
    Task(func() (string, error) {
        time.Sleep(2 * time.Second)
        return "step1", nil
    }).Named("slow-step1"),
    Task(func() (int, error) {
        time.Sleep(2 * time.Second)
        return 42, nil
    }).Named("slow-step2"),
    Task(func() (bool, error) {
        time.Sleep(2 * time.Second) // This will timeout
        return true, nil
    }).Named("slow-step3"),
).Named("timeout-test")

result, err := seq.Execute(ctx, config)
if err != nil {
    // Error includes timeout information and partial results
    log.Printf("Pipeline timed out: %v", err)
    
    // Check which steps completed before timeout
    if result.Get("slow-step1") != nil {
        log.Println("Step 1 completed before timeout")
    }
    if result.Get("slow-step2") != nil {
        log.Println("Step 2 completed before timeout")
    }
}
```

### Manual Cancellation

```go
ctx, cancel := context.WithCancel(context.Background())

// Start sequential execution in goroutine
go func() {
    result, err := seq.Execute(ctx, config)
    if err != nil {
        log.Printf("Pipeline cancelled: %v", err)
    }
}()

// Cancel after some condition
time.Sleep(1 * time.Second)
cancel() // This will cancel the sequential execution
```

## Performance Considerations

### Benchmarking Error Handling

```go
func BenchmarkErrorHandling(b *testing.B) {
    ctx := context.Background()
    config := Config{ErrorStrategy: FailFast}
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        seq := Sequential(
            Task(func() (int, error) { return 1, nil }),
            Task(func() (int, error) { return 0, errors.New("test error") }),
        ).Named("benchmark-test")
        
        result, err := seq.Execute(ctx, config)
        if err == nil {
            b.Fatal("Expected error")
        }
        if result == nil {
            b.Fatal("Expected result")
        }
    }
}
```

### Memory Efficiency

The enhanced error handling system is designed for minimal memory overhead:

- Error contexts are created only when needed
- Stack traces are captured efficiently using runtime.Stack
- Error boundaries use atomic operations for thread safety
- Object pooling reduces garbage collection pressure

## Best Practices

### 1. Choose Appropriate Error Strategies

```go
// Use FailFast for critical operations where any failure should stop execution
criticalPipeline := Sequential(tasks...).ErrorBoundary(FailFast)

// Use CollectAll for operations where you want maximum completion
notificationPipeline := Sequential(tasks...).ErrorBoundary(CollectAll)
```

### 2. Provide Meaningful Names

```go
// Good: Descriptive names help with debugging
seq := Sequential(
    Task(fetchUserData).Named("fetch-user-profile"),
    Task(validateUserData).Named("validate-user-permissions"),
    Task(processUserData).Named("process-user-request"),
).Named("user-request-pipeline")

// Bad: Generic names provide little debugging value
seq := Sequential(
    Task(fetchUserData).Named("task1"),
    Task(validateUserData).Named("task2"),
    Task(processUserData).Named("task3"),
).Named("pipeline")
```

### 3. Handle Errors Appropriately

```go
result, err := seq.Execute(ctx, config)
if err != nil {
    // Log the detailed error report
    log.Printf("Pipeline failed: %v", err)
    
    // Check for partial results
    if result != nil {
        // Process any successful results
        if userData := result.Get("fetch-user"); userData != nil {
            // Handle partial success
        }
        
        // Analyze specific errors
        for _, opErr := range result.Errors() {
            if strings.Contains(opErr.Error.Error(), "timeout") {
                // Handle timeout errors specifically
            }
        }
    }
    
    return err
}
```

### 4. Use Context Appropriately

```go
// Set reasonable timeouts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Pass context values for tracing
ctx = context.WithValue(ctx, "request-id", requestID)
ctx = context.WithValue(ctx, "user-id", userID)

result, err := seq.Execute(ctx, config)
```

This enhanced error handling system provides military-grade reliability while maintaining excellent performance and comprehensive observability for debugging and monitoring.