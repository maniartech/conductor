# Orchestrator - Military-Grade Goroutine Orchestration Library

[![Build Status](https://travis-ci.com/maniartech/orchestrator.svg?branch=master)](https://travis-ci.com/maniartech/orchestrator)

A high-performance, thread-safe goroutine orchestration library for Go with zero-allocation execution, comprehensive error handling, and complex nested orchestration support.

## 🚧 We are revamping this library dont use it yet

## Features

- 🚀 **Zero-allocation execution** - Optimized for high-performance scenarios
- 🔒 **Thread-safe** - All operations use atomic primitives and proper synchronization
- 🎯 **Generic type support** - Type-safe task execution with Go generics
- 🛡️ **Comprehensive error handling** - Panic recovery, timeout handling, and detailed error reporting
- 🔄 **Fluent API** - Easy-to-use builder pattern for configuration
- 📊 **Rich observability** - Status tracking, timing information, and stack traces
- 🏗️ **Modular architecture** - Clean separation of concerns with internal packages

## Quick Start

```bash
go get github.com/maniartech/orchestrator
```

### Simple Task Execution

```go
package main

import (
    "context"
    "github.com/maniartech/orchestrator"
)

func main() {
    // Simple task execution
    result, err := orchestrator.Setup(
        orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
            return "Hello, World!", nil
        }).Named("greeting"),
    ).Await()
    
    if err != nil {
        panic(err)
    }
    
    greeting := result.Get("greeting").(string)
    fmt.Println(greeting) // Output: Hello, World!
}
```

### Working Examples

Here are the currently working examples with the Task execution engine:

#### Basic Task with Configuration
```go
import "github.com/maniartech/orchestrator"

func processData() (int, error) {
    time.Sleep(10 * time.Millisecond)
    return 42, nil
}

func main() {
    result, err := orchestrator.Setup(
        orchestrator.Task(processData).
            Named("data-processor").
            With(orchestrator.Config{
                Timeout: 30 * time.Second,
            }),
    ).Await()
    
    if err != nil {
        panic(err)
    }
    
    answer := result.Get("data-processor").(int)
    fmt.Printf("Result: %d\n", answer)
}
```

#### Real-time Status Monitoring
```go
func monitoredExecution() {
    workflow := orchestrator.Setup(
        orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
            time.Sleep(100 * time.Millisecond)
            return "completed", nil
        }).Named("slow-task"),
    )
    
    // Monitor status in real-time
    go func() {
        for {
            status := workflow.GetStatus()
            fmt.Printf("Status: %s\n", status)
            if status == orchestrator.Completed || status == orchestrator.Cancelled {
                break
            }
            time.Sleep(10 * time.Millisecond)
        }
    }()
    
    result, err := workflow.Await()
    // Handle result...
}
```

### Coming Soon: Complex Resource Processing Orchestration

The following advanced orchestration capabilities are being implemented in the next development phases:

```go
// 🚧 COMING SOON - Sequential and Concurrent orchestrations (Tasks 4.1 & 5.1)
// This will provide the same expressive power as the legacy code but with better performance

import "github.com/maniartech/orchestrator"

// HandleResource will process various activities on the specified resource.
// All activities will be executed in their own goroutines in an orchestrated manner.
// This will provide concurrent, faster yet controlled execution.
//
//             |-----Task------------------|     |-----Task----|
//             |                           |     |             |
// ----Sequential----Concurrent----Sequential->>-Task->>-Task--|----Concurrent----Task----|----Await----
//             |                           |     |             |
//             |      |-----Task----|      |     |-----Task----|
//             |      |             |      |
//             |-----Concurrent----Task----|------|
//                    |             |
//                    |-----Task----|
//
func HandleResource(resourceId int) error {
    return orchestrator.Setup(
        orchestrator.Sequential(
            // Infrastructure preparation
            orchestrator.Task(keepInfraReady).Named("infra-ready"),
            
            // Concurrent resource processing
            orchestrator.Concurrent(
                // Main resource processing pipeline
                orchestrator.Sequential(
                    orchestrator.Task(func(ctx orchestrator.Context) error { return fetchResource(resourceId) }).Named("fetch-resource"),
                    orchestrator.Task(processResource).Named("process-resource"),
                    orchestrator.Task(submitResource).Named("submit-resource"),
                ).Named("resource-pipeline"),
                
                // Dependency preparation (concurrent)
                orchestrator.Concurrent(
                    orchestrator.Task(prepareDependencyA).Named("prep-dep-a"),
                    orchestrator.Task(prepareDependencyB).Named("prep-dep-b"),
                    orchestrator.Task(prepareDependencyC).Named("prep-dep-c"),
                ).Named("dependency-prep"),
            ).Named("main-processing"),
            
            // Final notifications (concurrent)
            orchestrator.Concurrent(
                orchestrator.Task(postToSocialMedia).Named("social-media"),
                orchestrator.Task(sendNotifications).Named("notifications"),
                orchestrator.Task(submitReport).Named("report"),
            ).Named("notifications"),
        ).Named("resource-handler"),
    ).Await()
}
```

### Advanced Task Configuration

Currently available with the Task execution engine:

```go
import (
    "time"
    "github.com/maniartech/orchestrator"
)

func advancedTaskExample() error {
    // Task with comprehensive configuration
    result, err := orchestrator.Setup(
        orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
            // Simulate complex processing
            time.Sleep(50 * time.Millisecond)
            return "processed", nil
        }).Named("complex-processor").
           With(orchestrator.Config{
               Timeout: 30 * time.Second,
               MaxConcurrency: 5,
           }).
           ErrorBoundary(orchestrator.CollectAll),
    ).With(orchestrator.Config{
        Timeout: 60 * time.Second, // Workflow-level timeout
    }).Await()
    
    if err != nil {
        return err
    }
    
    processed := result.Get("complex-processor").(string)
    fmt.Printf("Result: %s\n", processed)
    return nil
}
```

### Coming Soon: Advanced Configuration with Sequential/Concurrent

```go
// 🚧 COMING SOON - Advanced error handling and configuration (Tasks 4.2 & 5.2)
func HandleResourceAdvanced(resourceId int) error {
    return orchestrator.Setup(
        orchestrator.Sequential(
            // Infrastructure with custom timeout
            orchestrator.Task(keepInfraReady).
                Named("infra-ready").
                With(orchestrator.Config{Timeout: 30 * time.Second}),
            
            // Concurrent processing with different error strategies
            orchestrator.Concurrent(
                // Critical path - fail fast
                orchestrator.Sequential(
                    orchestrator.Task(func(ctx orchestrator.Context) error { return fetchResource(resourceId) }).
                        Named("fetch-resource").
                        With(orchestrator.Config{
                            Timeout: 10 * time.Second,
                            Retries: 3,
                        }),
                    orchestrator.Task(processResource).
                        Named("process-resource").
                        ErrorBoundary(orchestrator.FailFast),
                ).Named("critical-path").
                  ErrorBoundary(orchestrator.FailFast),
                
                // Non-critical dependencies - collect all errors
                orchestrator.Concurrent(
                    orchestrator.Task(prepareDependencyA).Named("prep-dep-a"),
                    orchestrator.Task(prepareDependencyB).Named("prep-dep-b"),
                    orchestrator.Task(prepareDependencyC).Named("prep-dep-c"),
                ).Named("dependency-prep").
                  ErrorBoundary(orchestrator.CollectAll),
            ).Named("main-processing"),
        ).Named("resource-handler"),
    ).Await()
}
```

### Current Status Monitoring Capabilities

```go
func monitorTaskExecution() error {
    workflow := orchestrator.Setup(
        orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
            // Simulate long-running task
            time.Sleep(200 * time.Millisecond)
            return "processing complete", nil
        }).Named("long-running-task"),
    )
    
    // Monitor status in real-time
    go func() {
        for {
            status := workflow.GetStatus()
            name := workflow.GetName()
            fmt.Printf("Task '%s' status: %s\n", name, status)
            
            if status == orchestrator.Completed || status == orchestrator.Cancelled {
                break
            }
            time.Sleep(50 * time.Millisecond)
        }
    }()
    
    result, err := workflow.Await()
    if err != nil {
        return err
    }
    
    fmt.Printf("Final result: %s\n", result.Get("long-running-task"))
    return nil
}
```

### Status Tracking

```go
// Check task status
fmt.Printf("Status: %s\n", task.GetStatus()) // NotStarted, Running, Completed, or Cancelled

// Get task configuration
config := task.GetConfig()
fmt.Printf("Timeout: %s\n", config.Timeout)
```

## Architecture

The orchestrator is built with a modular architecture:

```
internal/
├── task/           # Task execution engine with atomic status management
├── orchestration/  # Core interfaces and orchestration types
├── config/         # Configuration management with inheritance
├── errors/         # Error handling and reporting
├── result/         # Result management and type safety
├── context/        # Context management for inter-task communication
├── pool/           # Object pooling for performance optimization
└── status/         # Status management utilities
```

## Performance

- **Zero-allocation status operations** using atomic primitives
- **Minimal memory overhead** with object pooling
- **Lock-free status management** for maximum concurrency
- **Efficient goroutine lifecycle management**

## Thread Safety

All operations are thread-safe:
- Atomic status management using `sync/atomic`
- Concurrent access to task properties
- Race-condition free execution
- Proper resource cleanup

## Error Handling

Comprehensive error handling includes:
- **Panic recovery** with full stack traces
- **Timeout handling** with graceful termination
- **Context cancellation** support
- **Rich error metadata** with timing and operation IDs

## Legacy Code

The original orchestrator implementation has been moved to the `legacy/` directory. The new implementation provides:
- Better thread safety (no race conditions)
- Improved performance with zero-allocation operations
- Cleaner architecture with proper separation of concerns
- Enhanced error handling and observability

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests with 100% coverage
5. Ensure all tests pass with `go test -race`
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Status

This library is under active development as part of a comprehensive redesign. The new implementation focuses on:
- Military-grade reliability and performance
- Zero-allocation execution paths
- Comprehensive testing and documentation
- Thread-safe operations throughout

See [ROADMAP.md](ROADMAP.md) for development progress and future plans.