# Sequential Error Handling Guide

This guide provides comprehensive documentation for error handling patterns and best practices in the sequential orchestration package.

## Overview

The sequential orchestration package provides military-grade error handling with two primary strategies:

- **FailFast**: Stops execution immediately on the first error
- **CollectAll**: Continues execution and collects all errors for comprehensive reporting

## Error Handling Strategies

### FailFast Strategy

The FailFast strategy provides immediate error detection and minimal resource usage:

```go
seq := Sequential(
    Task(fetchUser),
    Task(validateUser),    // If this fails, execution stops here
    Task(processUser),     // This won't execute
).ErrorBoundary(errors.FailFast)

result, err := seq.Execute(ctx, config)
if err != nil {
    // Contains detailed error report with execution context
    log.Printf("Pipeline failed: %v", err)
}
```

**Characteristics:**
- Immediate termination on first error
- Minimal resource consumption
- Fast failure detection
- Partial results available for completed steps

**Use Cases:**
- Critical pipelines where any failure invalidates the entire process
- Resource-constrained environments
- Fast feedback requirements
- Dependency chains where later steps depend on earlier ones

### CollectAll Strategy

The CollectAll strategy provides comprehensive error visibility:

```go
seq := Sequential(
    Task(sendEmail),       // May fail, but continue
    Task(logAnalytics),    // May fail, but continue  
    Task(updateCache),     // May fail, but continue
).ErrorBoundary(errors.CollectAll)

result, err := seq.Execute(ctx, config)
if err != nil {
    // Contains all errors that occurred
    for _, opErr := range result.Errors() {
        log.Printf("Step %d failed: %v", opErr.Index, opErr.Error)
    }
}
```

**Characteristics:**
- Continues execution despite errors
- Collects all errors for comprehensive reporting
- Maximum operation completion
- Rich error context for debugging

**Use Cases:**
- Best-effort operations where partial success is valuable
- Batch processing where individual failures shouldn't stop the batch
- Comprehensive error reporting requirements
- Independent operations that don't depend on each other

## Error Boundary Containment

Error boundaries provide sophisticated error isolation and propagation control:

```go
// Critical operations with FailFast boundary
criticalSection := Sequential(
    Task(authenticate),
    Task(authorize),
).Named("critical-section").ErrorBoundary(errors.FailFast)

// Optional operations with CollectAll boundary
optionalSection := Sequential(
    Task(sendWelcomeEmail),
    Task(logAnalytics),
).Named("optional-section").ErrorBoundary(errors.CollectAll)

// Main pipeline
mainPipeline := Sequential(
    criticalSection,
    optionalSection,
).Named("main-pipeline").ErrorBoundary(errors.FailFast)
```

**Error Boundary Rules:**
1. Errors are contained within their boundary scope
2. Child boundaries override parent error strategies
3. Error propagation follows the boundary hierarchy
4. Each boundary provides its own error context

## Rich Error Metadata

Every error includes comprehensive metadata for debugging and observability:

```go
type OperationError struct {
    Error     error         // Original error
    Index     int           // Step index where error occurred
    Duration  time.Duration // How long the step took to fail
    Timestamp time.Time     // When the error occurred
    OpID      string        // Unique operation identifier
    Stack     []byte        // Stack trace (for panics)
}
```

**Accessing Error Metadata:**
```go
result, err := seq.Execute(ctx, config)
if err != nil {
    for _, opErr := range result.Errors() {
        log.Printf("Operation %s failed at index %d after %v: %v",
            opErr.OpID, opErr.Index, opErr.Duration, opErr.Error)
        
        if len(opErr.Stack) > 0 {
            log.Printf("Stack trace: %s", string(opErr.Stack))
        }
    }
}
```

## Configuration Inheritance

Error handling strategies support hierarchical configuration with local overrides:

```go
// Parent configuration
parentConfig := config.Config{
    ErrorStrategy: errors.FailFast,
    Timeout:       30 * time.Second,
}

// Child overrides error strategy but inherits timeout
seq := Sequential(tasks...).
    With(config.Config{ErrorStrategy: errors.CollectAll})

// Child uses CollectAll strategy with 30-second timeout
result, err := seq.Execute(ctx, parentConfig)
```

**Inheritance Rules:**
1. Child configurations override parent settings
2. Unspecified child settings inherit from parent
3. Error boundaries take precedence over configuration
4. Local overrides apply only to the specific orchestration

## Panic Recovery

The sequential orchestration automatically recovers from panics and converts them to errors:

```go
seq := Sequential(
    Task(func() (string, error) { return "step1", nil }),
    Task(func() (string, error) { panic("something went wrong") }),
    Task(func() (string, error) { return "step3", nil }),
)

result, err := seq.Execute(ctx, config)
// Panic is recovered and converted to an error
// Stack trace is captured in error metadata
```

**Panic Recovery Features:**
- Automatic panic detection and recovery
- Stack trace capture for debugging
- Conversion to standard error format
- Integration with error boundary containment

## Best Practices

### 1. Choose the Right Strategy

**Use FailFast when:**
- Operations are dependent on each other
- Fast failure detection is critical
- Resource conservation is important
- Any failure invalidates the entire process

**Use CollectAll when:**
- Operations are independent
- Partial success has value
- Comprehensive error reporting is needed
- Best-effort execution is acceptable

### 2. Design Error Boundaries Thoughtfully

```go
// Good: Separate critical and optional operations
pipeline := Sequential(
    // Critical operations that must succeed
    Sequential(
        Task(authenticate),
        Task(validateInput),
    ).Named("critical").ErrorBoundary(errors.FailFast),
    
    // Optional operations that can fail
    Sequential(
        Task(sendNotification),
        Task(updateAnalytics),
    ).Named("optional").ErrorBoundary(errors.CollectAll),
)
```

### 3. Use Meaningful Names for Observability

```go
// Good: Descriptive names for debugging
seq := Sequential(
    Task(fetchUserData).Named("fetch-user"),
    Task(validateUser).Named("validate-user"),
    Task(processUser).Named("process-user"),
).Named("user-processing-pipeline")
```

### 4. Handle Context Cancellation Gracefully

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := seq.Execute(ctx, config)
if errors.Is(err, context.DeadlineExceeded) {
    log.Printf("Pipeline timed out after 30 seconds")
    // Handle timeout-specific logic
}
```

### 5. Leverage Error Metadata for Monitoring

```go
result, err := seq.Execute(ctx, config)
if err != nil {
    for _, opErr := range result.Errors() {
        // Send metrics to monitoring system
        metrics.RecordError(opErr.OpID, opErr.Duration, opErr.Error)
        
        // Log structured error information
        logger.WithFields(map[string]interface{}{
            "operation_id": opErr.OpID,
            "step_index":   opErr.Index,
            "duration_ms":  opErr.Duration.Milliseconds(),
            "error":        opErr.Error.Error(),
        }).Error("Sequential step failed")
    }
}
```

## Performance Considerations

### Zero-Allocation Error Collection

The error handling system is designed for high performance:

- Atomic operations for error counting
- Pre-allocated error slices where possible
- Minimal memory allocations during error collection
- Lock-free error metadata access

### Benchmarking Error Handling

```go
func BenchmarkErrorHandling(b *testing.B) {
    ctx := context.Background()
    cfg := config.Config{ErrorStrategy: errors.FailFast}
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        seq := Sequential(
            Task(func() (int, error) { return 1, nil }),
            Task(func() (int, error) { return 0, errors.New("test") }),
        )
        
        result, err := seq.Execute(ctx, cfg)
        // Process result and error
    }
}
```

## Advanced Patterns

### Retry with Error Boundaries

```go
// Implement retry logic within error boundaries
retryableTask := Sequential(
    Task(func() (string, error) {
        // Attempt operation with potential for transient failure
        return attemptOperation()
    }),
).Named("retryable-operation").ErrorBoundary(errors.CollectAll)

// Wrap in retry logic
for attempt := 0; attempt < maxRetries; attempt++ {
    result, err := retryableTask.Execute(ctx, config)
    if err == nil {
        break // Success
    }
    
    if attempt < maxRetries-1 {
        time.Sleep(backoffDelay)
        backoffDelay *= 2 // Exponential backoff
    }
}
```

### Circuit Breaker Pattern

```go
type CircuitBreaker struct {
    failures    int64
    lastFailure time.Time
    threshold   int
    timeout     time.Duration
}

func (cb *CircuitBreaker) Execute(operation func() (interface{}, error)) (interface{}, error) {
    if atomic.LoadInt64(&cb.failures) >= int64(cb.threshold) {
        if time.Since(cb.lastFailure) < cb.timeout {
            return nil, errors.New("circuit breaker open")
        }
        atomic.StoreInt64(&cb.failures, 0) // Reset
    }
    
    result, err := operation()
    if err != nil {
        atomic.AddInt64(&cb.failures, 1)
        cb.lastFailure = time.Now()
    }
    
    return result, err
}
```

### Error Aggregation and Reporting

```go
type ErrorAggregator struct {
    errors map[string][]errors.OperationError
    mu     sync.RWMutex
}

func (ea *ErrorAggregator) CollectErrors(pipelineName string, result *result.Result) {
    ea.mu.Lock()
    defer ea.mu.Unlock()
    
    if ea.errors == nil {
        ea.errors = make(map[string][]errors.OperationError)
    }
    
    ea.errors[pipelineName] = append(ea.errors[pipelineName], result.Errors()...)
}

func (ea *ErrorAggregator) GenerateReport() string {
    ea.mu.RLock()
    defer ea.mu.RUnlock()
    
    var report strings.Builder
    for pipeline, errs := range ea.errors {
        report.WriteString(fmt.Sprintf("Pipeline: %s (%d errors)\n", pipeline, len(errs)))
        for _, err := range errs {
            report.WriteString(fmt.Sprintf("  - %s: %v\n", err.OpID, err.Error))
        }
    }
    return report.String()
}
```

## Testing Error Handling

### Unit Testing Error Strategies

```go
func TestErrorStrategy(t *testing.T) {
    tests := []struct {
        name     string
        strategy errors.ErrorStrategy
        wantSteps int
        wantErrors int
    }{
        {"FailFast", errors.FailFast, 2, 1},
        {"CollectAll", errors.CollectAll, 4, 2},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var executedSteps int64
            
            seq := Sequential(
                Task(func() (string, error) {
                    atomic.AddInt64(&executedSteps, 1)
                    return "step1", nil
                }),
                Task(func() (string, error) {
                    atomic.AddInt64(&executedSteps, 1)
                    return "", errors.New("error1")
                }),
                Task(func() (string, error) {
                    atomic.AddInt64(&executedSteps, 1)
                    return "step3", nil
                }),
                Task(func() (string, error) {
                    atomic.AddInt64(&executedSteps, 1)
                    return "", errors.New("error2")
                }),
            ).ErrorBoundary(tt.strategy)
            
            result, err := seq.Execute(context.Background(), config.Config{})
            
            if int(atomic.LoadInt64(&executedSteps)) != tt.wantSteps {
                t.Errorf("Expected %d steps executed, got %d", 
                    tt.wantSteps, atomic.LoadInt64(&executedSteps))
            }
            
            if len(result.Errors()) != tt.wantErrors {
                t.Errorf("Expected %d errors, got %d", 
                    tt.wantErrors, len(result.Errors()))
            }
        })
    }
}
```

### Integration Testing with Real Scenarios

```go
func TestRealWorldErrorScenario(t *testing.T) {
    // Simulate a real-world data processing pipeline
    pipeline := Sequential(
        Task(loadData).Named("load-data"),
        Task(validateData).Named("validate-data"),
        Task(transformData).Named("transform-data"),
        Task(saveData).Named("save-data"),
    ).Named("data-pipeline").ErrorBoundary(errors.CollectAll)
    
    // Test with various failure scenarios
    scenarios := []struct {
        name           string
        injectFailures []int // Step indices to fail
        expectResults  []string
    }{
        {"AllSuccess", []int{}, []string{"loaded", "validated", "transformed", "saved"}},
        {"LoadFailure", []int{0}, []string{}},
        {"ValidateFailure", []int{1}, []string{"loaded"}},
        {"MultipleFailures", []int{1, 3}, []string{"loaded", "transformed"}},
    }
    
    for _, scenario := range scenarios {
        t.Run(scenario.name, func(t *testing.T) {
            // Configure failure injection
            configureFailures(scenario.injectFailures)
            
            result, err := pipeline.Execute(context.Background(), config.Config{})
            
            // Verify expected behavior
            if len(scenario.injectFailures) > 0 && err == nil {
                t.Error("Expected error when failures are injected")
            }
            
            // Verify partial results
            for _, expectedResult := range scenario.expectResults {
                if result.Get(expectedResult) == nil {
                    t.Errorf("Expected result %s not found", expectedResult)
                }
            }
        })
    }
}
```

## Troubleshooting

### Common Issues and Solutions

1. **Memory Leaks in Error Collection**
   - Ensure error slices are properly sized
   - Use object pooling for frequently created orchestrations
   - Monitor memory usage in long-running applications

2. **Performance Degradation with Many Errors**
   - Consider using FailFast for performance-critical paths
   - Implement error sampling for high-volume scenarios
   - Use structured logging instead of detailed error reports

3. **Context Cancellation Not Respected**
   - Check that tasks properly handle context cancellation
   - Ensure context is passed correctly through the call chain
   - Use context.WithTimeout for time-bounded operations

4. **Error Boundary Confusion**
   - Document error boundary design decisions
   - Use clear naming for different boundary scopes
   - Test error propagation behavior thoroughly

### Debugging Tips

1. **Enable Detailed Error Reports**
   ```go
   // Enhanced error reporting provides detailed context
   result, err := seq.Execute(ctx, config)
   if err != nil {
       log.Printf("Detailed error report:\n%s", err.Error())
   }
   ```

2. **Use Operation IDs for Tracing**
   ```go
   for _, opErr := range result.Errors() {
       log.Printf("Trace operation %s: %v", opErr.OpID, opErr.Error)
   }
   ```

3. **Monitor Error Patterns**
   ```go
   errorCounts := make(map[string]int)
   for _, opErr := range result.Errors() {
       errorCounts[opErr.OpID]++
   }
   ```

This comprehensive guide covers all aspects of error handling in the sequential orchestration package, providing both theoretical understanding and practical implementation guidance.