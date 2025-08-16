# Enhanced Context Design

## Overview

The enhanced context system provides enterprise-grade orchestration context management with:
- **Enhanced timeout management** using atomic operations for zero-allocation hot paths
- **Graceful shutdown coordination** with hierarchical cleanup
- **Nested orchestration lifecycle** for complex workflow hierarchies

## Design Principles

### 1. KISS (Keep It Simple, Stupid)
- Single responsibility per method
- Minimal interface surface area
- Clear separation of concerns

### 2. Single Responsibility Principle (SRP)
- Context = data storage and communication only
- Configuration = execution behavior and policies
- Orchestrator = actual retry/timeout enforcement

### 3. Zero-Allocation Hot Paths
- `IsExpired()` - atomic load only
- `GetRemainingTime()` - atomic calculation
- `IsShuttingDown()` - atomic load only
- `GetPath()` - cached string access

### 4. Enterprise Requirements
- Distributed tracing support via hierarchical paths
- Automatic resource cleanup via parent-child relationships
- Security context inheritance
- Performance isolation via hierarchical timeouts

## Interface Design

```go
type Context interface {
    // Core operations (existing)
    Get(name string) any
    Set(name string, value any)
    Config() config.Config
    
    // Lifecycle management
    Cancel()
    Done() <-chan struct{}
    Err() error
    
    // Enhanced timeout management
    WithTimeout(timeout time.Duration) Context
    WithDeadline(deadline time.Time) Context
    GetRemainingTime() time.Duration
    IsExpired() bool
    
    // Nested orchestration lifecycle
    CreateChild(name string) Context
    GetParent() Context
    GetPath() string
    
    // Graceful shutdown coordination
    Shutdown() <-chan struct{}
    IsShuttingDown() bool
}
```

## Implementation Architecture

### 1. Atomic Timeout Management

```go
type contextImpl struct {
    // Existing fields...
    
    // Timeout management (atomic for zero-allocation)
    deadline     atomic.Int64  // Unix nanoseconds, 0 = no timeout
    timeoutSet   atomic.Bool   // Whether timeout is active
    
    // Timeout monitoring
    timeoutTimer *time.Timer
    timerMu      sync.Mutex
}
```

**Key Features:**
- Atomic operations for `IsExpired()` and `GetRemainingTime()`
- Single timer per context for efficiency
- Automatic cleanup when context is cancelled

### 2. Hierarchical Context Structure

```go
type contextImpl struct {
    // Existing fields...
    
    // Hierarchy (cached for performance)
    parent   *contextImpl
    children []*contextImpl
    childMu  sync.RWMutex
    name     string
    path     string  // Cached: "/parent/child/grandchild"
    depth    int     // Cached: 0, 1, 2, ...
}
```

**Key Features:**
- Cached path and depth for zero-allocation queries
- Parent-child relationships for cleanup propagation
- Thread-safe child management

### 3. Shutdown Coordination

```go
type contextImpl struct {
    // Existing fields...
    
    // Shutdown management (atomic)
    shutdownInitiated atomic.Bool
    shutdownChan      chan struct{}  // Pre-allocated
}
```

**Key Features:**
- Atomic shutdown status checking
- Pre-allocated channels to avoid allocation during shutdown
- Hierarchical shutdown propagation

## Usage Examples

### 1. Basic Timeout Management

```go
// Create context with timeout
ctx := NewContext(config.DefaultConfig())
ctxWithTimeout := ctx.WithTimeout(30 * time.Second)

// Zero-allocation timeout checking (hot path)
if ctxWithTimeout.IsExpired() {
    return errors.New("operation timed out")
}

// Get remaining time for progress reporting
remaining := ctxWithTimeout.GetRemainingTime()
fmt.Printf("Time remaining: %v\n", remaining)
```

### 2. Nested Orchestration Hierarchy

```go
// Root context: API request
apiCtx := ctx.CreateChild("api-request")
apiCtx.Set("request_id", "REQ-12345")
apiCtx.Set("user_id", "USR-67890")

// Child context: Database operations
dbCtx := apiCtx.CreateChild("database-ops")
dbCtx.Set("transaction_id", "TXN-98765")

// Grandchild context: Specific query
queryCtx := dbCtx.CreateChild("user-lookup")

// Distributed tracing path
fmt.Println(queryCtx.GetPath()) // "/api-request/database-ops/user-lookup"

// Parent access for cleanup
parent := queryCtx.GetParent() // Returns dbCtx
```

### 3. Graceful Shutdown

```go
// Monitor shutdown status (zero-allocation)
if ctx.IsShuttingDown() {
    return errors.New("system is shutting down")
}

// Wait for shutdown signal
select {
case <-ctx.Shutdown():
    // Perform cleanup
    return nil
case <-time.After(timeout):
    // Continue operation
}
```

### 4. Enterprise Workflow Example

```go
// Root context: Order processing
orderCtx := ctx.CreateChild("order-processing")
orderCtx.Set("order_id", "ORD-12345")
orderCtx.Set("customer_tier", "premium")

// Child context: Payment with timeout
paymentCtx := orderCtx.CreateChild("payment-processing").WithTimeout(30 * time.Second)
paymentCtx.Set("payment_method", "credit_card")
paymentCtx.Set("requires_3ds", true)

// Child context: Inventory check
inventoryCtx := orderCtx.CreateChild("inventory-check").WithTimeout(10 * time.Second)
inventoryCtx.Set("warehouse_id", "WH-001")

// Each context has its own timeout and inherits parent data
// Path: "/order-processing/payment-processing" and "/order-processing/inventory-check"
```

## Performance Characteristics

### Zero-Allocation Operations
- `IsExpired()`: Single atomic load
- `GetRemainingTime()`: Atomic load + arithmetic
- `IsShuttingDown()`: Single atomic load
- `GetPath()`: Cached string access
- `GetParent()`: Pointer access

### Memory Efficiency
- Pre-allocated channels for shutdown coordination
- Cached path strings to avoid string building
- Atomic operations instead of mutexes where possible
- Minimal struct overhead per context

### Concurrency Safety
- All timeout operations are atomic
- Thread-safe child management with RWMutex
- Lock-free status checking for hot paths
- Proper synchronization for hierarchy modifications

## Error Handling

### Timeout Errors
- Contexts automatically cancel when timeout expires
- `Err()` method returns `context.DeadlineExceeded`
- Parent contexts are not affected by child timeouts

### Shutdown Errors
- Graceful shutdown propagates to all children
- Operations can check shutdown status before starting
- Clean resource cleanup via context cancellation

### Hierarchy Errors
- Invalid parent-child relationships are prevented
- Circular references are impossible by design
- Memory leaks prevented by proper cleanup

## Testing Strategy

### Unit Tests
- All public methods with 100% coverage
- Edge cases: expired timeouts, shutdown scenarios
- Concurrent access patterns with race detection
- Memory allocation testing (zero-alloc verification)

### Integration Tests
- Complex nested hierarchies
- Timeout propagation scenarios
- Shutdown coordination across multiple levels
- Real-world enterprise workflow patterns

### Performance Tests
- Benchmark zero-allocation operations
- Load testing with thousands of contexts
- Memory stability over long periods
- Concurrent access performance

## Migration from Current Implementation

### Backward Compatibility
- All existing Context interface methods remain unchanged
- New methods are additive only
- Existing code continues to work without modification

### New Capabilities
- Enhanced timeout management with atomic operations
- Nested context hierarchies for complex workflows
- Graceful shutdown coordination
- Enterprise-grade observability support

This design provides enterprise-grade capabilities while maintaining KISS principles and zero-allocation performance for critical paths.