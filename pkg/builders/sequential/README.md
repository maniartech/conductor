# Sequential Orchestration Examples

This directory contains comprehensive examples demonstrating the sequential orchestration capabilities of the orchestrator library. The examples progress from simple to complex, showcasing all features and best practices.

## Running the Examples

To run all example tests:
```bash
go test ./internal/sequential -run="TestExample" -v
```

To run a specific example:
```bash
go test ./internal/sequential -run="TestExample_SimpleSequential" -v
```

To run performance benchmarks:
```bash
go test ./internal/sequential -bench=. -benchmem
```

## Example Categories

### 1. Basic Examples

#### Simple Sequential Orchestration (`TestExample_SimpleSequential`)
Demonstrates the most basic sequential orchestration with three simple steps.

**Features shown:**
- Basic Sequential constructor
- Named tasks
- Result collection
- Simple error handling

**Output:**
```
=== Example: Simple Sequential Orchestration ===
Step 1: Fetching user data...
Step 2: Validating user...
Step 3: Processing user...
✅ Simple pipeline completed successfully
   - User ID: user-123
   - Validation: validated
   - Processing: processed
```

#### Error Handling Strategies (`TestExample_ErrorHandling`)
Shows the difference between FailFast and CollectAll error strategies.

**Features shown:**
- FailFast strategy (stops on first error)
- CollectAll strategy (continues despite errors)
- Error result analysis
- Partial result collection

**Output:**
```
=== Example: Error Handling Strategies ===
--- Testing FailFast Strategy ---
❌ FailFast stopped at first error
   - Success step result: success
   - Skipped step result: <nil>
--- Testing CollectAll Strategy ---
⚠️  CollectAll completed with errors
   - Success results: success, true
   - Total errors collected: 1
```

### 2. Real-World Examples

#### E-commerce Order Processing (`TestExample_RealWorldEcommerce`)
A realistic e-commerce order processing pipeline with multiple business steps.

**Features shown:**
- Complex data structures
- Business logic simulation
- Timing measurements
- Comprehensive result logging
- Error handling in business context

**Steps:**
1. Order validation
2. Inventory checking
3. Payment processing
4. Shipment creation
5. Confirmation sending

**Output:**
```
=== Example: Real-World E-commerce Order Processing ===
🔍 Validating order...
   Order order-12345 validated successfully
📦 Checking inventory...
   All items in stock
💳 Processing payment...
   Payment processed: payment-order-12345
📮 Creating shipment...
   Shipment created: shipment-order-12345
📧 Sending confirmation...
   Confirmation sent: confirmation-order-12345
✅ Order processing completed successfully!
   Processing time: 97ms
```

#### Data Processing Pipeline (`TestExample_DataPipeline`)
Demonstrates a typical data processing workflow with transformation steps.

**Features shown:**
- Data structure transformations
- Multi-step data processing
- Performance timing
- Result validation
- Type-safe operations

**Steps:**
1. Load raw data
2. Validate data quality
3. Transform data format
4. Save processed results

**Output:**
```
=== Example: Data Processing Pipeline ===
📥 Loading raw data...
   Loaded 3 records
✅ Validating data...
   Validated 3/3 records
🔄 Transforming data...
   Transformed 3 records
💾 Saving results...
   Results saved with ID: save-20250807125303
✅ Data pipeline completed successfully in 107ms
```

### 3. Advanced Examples

#### Nested Orchestrations (`TestExample_NestedOrchestrations`)
Shows how to compose sequential orchestrations within other sequential orchestrations.

**Features shown:**
- Nested sequential composition
- Sub-pipeline organization
- Result aggregation from nested orchestrations
- Hierarchical naming

**Structure:**
```
Main Pipeline
├── Authentication Pipeline
│   ├── Validate credentials
│   └── Generate token
└── Data Processing Pipeline
    ├── Fetch user data
    └── Transform data
```

#### Context Cancellation (`TestExample_ContextCancellation`)
Demonstrates how sequential orchestrations handle context cancellation and timeouts.

**Features shown:**
- Context timeout handling
- Partial execution results
- Cancellation detection
- Resource cleanup

**Output:**
```
=== Example: Context Cancellation ===
Step 1: Quick task
Step 2: Long task (will be cancelled)
⏰ Pipeline cancelled after 100ms
   - Quick task completed: true
   - Long task completed: false
   - Never executed: false
```

#### Error Boundaries (`TestExample_ErrorBoundaries`)
Shows how to use error boundaries to control error propagation in different sections.

**Features shown:**
- Critical sections with FailFast
- Optional sections with CollectAll
- Error boundary isolation
- Mixed error strategies

**Structure:**
```
Main Pipeline
├── Critical Section (FailFast)
│   ├── Authentication
│   └── Permission validation
└── Optional Section (CollectAll)
    ├── Welcome email (may fail)
    └── Analytics logging
```

## Performance Characteristics

The sequential orchestration system is designed for high performance:

```
BenchmarkSequentialBuilder_Execute-16    199986    5591 ns/op    2531 B/op    34 allocs/op
```

**Performance metrics:**
- **Execution time**: ~5.6 microseconds per sequential execution
- **Memory usage**: 2.5KB per execution
- **Allocations**: Only 34 allocations per execution
- **Throughput**: ~179,000 sequential executions per second

## Key Features Demonstrated

### 1. Builder Pattern
```go
seq := Sequential(tasks...).
    Named("my-pipeline").
    With(config.Config{Timeout: 30*time.Second}).
    ErrorBoundary(errors.CollectAll)
```

### 2. Error Strategies
- **FailFast**: Stop on first error, minimal resource usage
- **CollectAll**: Continue execution, collect all errors

### 3. Configuration Inheritance
- Parent configurations are inherited by children
- Local configurations override parent settings
- Error boundaries provide fine-grained control

### 4. Rich Error Context
- Detailed error reports with timing information
- Stack traces for debugging
- Operation IDs for traceability
- Comprehensive error metadata

### 5. Context Management
- Timeout handling
- Cancellation support
- Resource cleanup
- Graceful degradation

### 6. Type Safety
- Generic task support
- Type-safe result retrieval
- Compile-time type checking
- Runtime type validation

## Best Practices Shown

### 1. Naming Convention
```go
// Good: Descriptive names
.Named("user-authentication-pipeline")
.Named("validate-payment-method")

// Bad: Generic names
.Named("pipeline1")
.Named("task2")
```

### 2. Error Strategy Selection
```go
// Critical operations: Use FailFast
criticalPipeline.ErrorBoundary(errors.FailFast)

// Optional operations: Use CollectAll
notificationPipeline.ErrorBoundary(errors.CollectAll)
```

### 3. Configuration Management
```go
// Set timeouts appropriately
.With(config.Config{
    Timeout: 30*time.Second,        // Reasonable timeout
    ErrorStrategy: errors.FailFast,  // Appropriate strategy
})
```

### 4. Result Handling
```go
result, err := seq.Execute(ctx, config)
if err != nil {
    // Handle errors appropriately
    log.Printf("Pipeline failed: %v", err)
    
    // Check for partial results
    if result != nil {
        // Process successful steps
        if userData := result.Get("fetch-user"); userData != nil {
            // Handle partial success
        }
    }
}
```

### 5. Context Usage
```go
// Set reasonable timeouts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Pass context values for tracing
ctx = context.WithValue(ctx, "request-id", requestID)
```

## Testing Strategy

The examples serve multiple purposes:

1. **Documentation**: Show how to use the library
2. **Testing**: Verify functionality works correctly
3. **Benchmarking**: Measure performance characteristics
4. **Validation**: Ensure examples stay current with API changes

Each example is a complete, runnable test that demonstrates real-world usage patterns and best practices.

## Integration with Main Library

These examples are designed to work with the main orchestrator library and demonstrate integration patterns with:

- Task orchestrations
- Configuration management
- Error handling systems
- Result collection
- Status management
- Context handling

The examples provide a comprehensive guide for developers to understand and effectively use the sequential orchestration capabilities.