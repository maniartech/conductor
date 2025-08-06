# Orchestrator Roadmap

## Phase 1: Core Implementation (Current)
- Simple `interface{}` based result storage
- Clean API with hierarchical configuration
- Zero-allocation orchestration execution
- Military-grade error handling and recovery
- Comprehensive test coverage

## Phase 2: Performance Optimizations (Future)

### Zero-Allocation Result Storage
**Goal:** Eliminate `interface{}` boxing allocations in result storage and retrieval.

**Approach:** Hybrid storage system with performance tiers:
- **Fast Path:** Common types (string, int, User, Order) stored in zero-allocation 64-byte stack storage
- **Slow Path:** Arbitrary types stored with minimal allocation using `interface{}`
- **Auto-routing:** Compile-time or build-time analysis determines optimal storage path

**Technical Details:**
```go
type Result struct {
    entries map[string]resultEntry
    mu      sync.RWMutex
}

type resultEntry struct {
    typeID uint8
    data   interface{} // Optimized based on typeID
}

// Zero-allocation for common types, minimal allocation for others
func (r *Result) GetTyped[T any](name string) (T, bool)
```

**Benefits:**
- 90% of use cases: Zero allocation
- 10% of use cases: Minimal allocation  
- Unlimited type support
- Backward compatible API

**Implementation Complexity:**
- Type classification system (build-time analysis)
- Discriminated union storage
- Tiered access patterns
- Additional testing complexity

**Decision Rationale:**
Deferred to Phase 2 to maintain KISS principle in initial implementation while providing clear optimization path for performance-critical users.

### Advanced Type System
- Code generation for type-safe accessors
- Build-time type analysis for optimization
- Custom serialization for complex types

### Enhanced Observability
- Detailed performance metrics
- Memory usage tracking
- Execution tracing integration

## Phase 3: Enterprise Features (Future)
- Distributed orchestration
- Persistent workflow state
- Advanced retry policies with circuit breakers
- Integration with monitoring systems

## Dynamic Task at Runtime

Allow tasks to be dynamically at  runtime.

## Other

- Throughly test library for race conditions using `go test -race`
