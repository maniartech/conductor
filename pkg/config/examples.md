# Configuration System Examples and Best Practices

This document provides comprehensive examples and best practices for using the orchestrator configuration system.

## Overview

The configuration system provides two main approaches:
1. **Direct Config Creation**: Using struct literals and the `Inherit` method
2. **Builder Pattern**: Using `ConfigBuilder` for fluent, validated configuration construction

## Basic Usage

### Using ConfigBuilder (Recommended)

```go
// Simple configuration
config := core.NewConfigBuilder().
    ErrorStrategy(core.CollectAll).
    Timeout(30 * time.Second).
    MaxConcurrency(100).
    MustBuild()

// With error handling
config, err := core.NewConfigBuilder().
    ErrorStrategy(core.FailFast).
    Timeout(60 * time.Second).
    Retries(3).
    Build()
if err != nil {
    log.Fatal(err)
}
```

### Using Direct Config Creation

```go
// Basic config
config := core.Config{
    ErrorStrategy:  core.CollectAll,
    Timeout:        30 * time.Second,
    MaxConcurrency: 100,
    Context:        context.Background(),
}

// With inheritance
parentConfig := core.DefaultConfig()
childConfig := core.Config{
    Timeout: 10 * time.Second, // Override timeout
}.Inherit(parentConfig)
```

## Advanced Patterns

### Configuration Templates

Create reusable configuration templates for common scenarios:

```go
// High-performance template
func HighPerformanceConfig() core.Config {
    return core.NewConfigBuilder().
        ErrorStrategy(core.FailFast).
        Timeout(5 * time.Second).
        MaxConcurrency(1000).
        Retries(0).
        MustBuild()
}

// Fault-tolerant template
func FaultTolerantConfig() core.Config {
    return core.NewConfigBuilder().
        ErrorStrategy(core.CollectAll).
        Timeout(60 * time.Second).
        MaxConcurrency(50).
        Retries(3).
        MustBuild()
}

// Usage
config := HighPerformanceConfig()
```

### Configuration Inheritance Chains

Build complex configuration hierarchies:

```go
// Base organizational config
orgConfig := core.NewConfigBuilder().
    ErrorStrategy(core.FailFast).
    Timeout(30 * time.Second).
    MaxConcurrency(100).
    MustBuild()

// Team-specific config
teamConfig := core.NewConfigBuilderFrom(orgConfig).
    Timeout(60 * time.Second). // Teams get longer timeouts
    MustBuild()

// Project-specific config
projectConfig := core.NewConfigBuilderFrom(teamConfig).
    ErrorStrategy(core.CollectAll). // This project needs error collection
    MaxConcurrency(200).            // This project needs higher concurrency
    MustBuild()
```

### Context-Aware Configuration

Integrate with Go's context system:

```go
// With timeout context
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

config := core.NewConfigBuilder().
    WithContext(ctx).
    ErrorStrategy(core.CollectAll).
    MustBuild()

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

config := core.NewConfigBuilder().
    WithContext(ctx).
    Timeout(30 * time.Second).
    MustBuild()

// Cancel operations externally
go func() {
    time.Sleep(10 * time.Second)
    cancel() // This will cancel all operations using this config
}()
```

## Error Handling Strategies

### Fail Fast Strategy

Use when you need immediate feedback and want to stop on the first error:

```go
config := core.NewConfigBuilder().
    ErrorStrategy(core.FailFast).
    Timeout(10 * time.Second).
    MustBuild()

// Good for:
// - Critical operations where any failure is unacceptable
// - Resource-intensive operations where continuing would waste resources
// - Operations with dependencies where later steps depend on earlier ones
```

### Collect All Strategy

Use when you want to gather all errors and continue processing:

```go
config := core.NewConfigBuilder().
    ErrorStrategy(core.CollectAll).
    Timeout(30 * time.Second).
    MustBuild()

// Good for:
// - Batch processing where you want to process as much as possible
// - Validation scenarios where you want to report all errors
// - Independent operations where one failure doesn't affect others
```

## Performance Optimization

### Zero-Allocation Configuration

The ConfigBuilder is designed for zero allocations during normal operation:

```go
// This produces zero heap allocations
config := core.NewConfigBuilder().
    ErrorStrategy(core.CollectAll).
    Timeout(30 * time.Second).
    MaxConcurrency(100).
    MustBuild()
```

### Configuration Reuse

Reuse configurations to avoid repeated construction:

```go
// Create once, reuse many times
baseConfig := core.NewConfigBuilder().
    ErrorStrategy(core.CollectAll).
    MaxConcurrency(100).
    MustBuild()

// Clone for variations
fastConfig := core.NewConfigBuilderFrom(baseConfig).
    Timeout(5 * time.Second).
    MustBuild()

slowConfig := core.NewConfigBuilderFrom(baseConfig).
    Timeout(60 * time.Second).
    MustBuild()
```

### Builder Reuse

Reuse builders for multiple configurations:

```go
builder := core.NewConfigBuilder().
    ErrorStrategy(core.CollectAll).
    MaxConcurrency(100)

// Create multiple configs from same base
config1 := builder.Clone().Timeout(10 * time.Second).MustBuild()
config2 := builder.Clone().Timeout(30 * time.Second).MustBuild()
config3 := builder.Clone().Timeout(60 * time.Second).MustBuild()
```

## Validation and Error Handling

### Comprehensive Validation

The ConfigBuilder provides comprehensive validation:

```go
config, err := core.NewConfigBuilder().
    ErrorStrategy(core.CollectAll).
    Timeout(-1 * time.Second). // Invalid!
    MaxConcurrency(0).          // Invalid!
    Build()

if err != nil {
    // Handle validation errors
    fmt.Printf("Configuration validation failed: %v\n", err)
    return
}
```

### Safe Configuration with MustBuild

Use `MustBuild()` when you're certain the configuration is valid:

```go
// This will panic if configuration is invalid
config := core.NewConfigBuilder().
    ErrorStrategy(core.CollectAll).
    Timeout(30 * time.Second).
    MaxConcurrency(100).
    MustBuild()
```

### Error Accumulation

The builder accumulates errors and reports them all at once:

```go
builder := core.NewConfigBuilder()
builder.Timeout(-1 * time.Second)    // Error 1
builder.MaxConcurrency(-1)           // Error 2
builder.Retries(-1)                  // Error 3

// All errors reported together
config, err := builder.Build()
if err != nil {
    // err contains all three validation errors
    fmt.Printf("Multiple errors: %v\n", err)
}
```

## Best Practices

### 1. Use ConfigBuilder for New Code

```go
// Preferred
config := core.NewConfigBuilder().
    ErrorStrategy(core.CollectAll).
    Timeout(30 * time.Second).
    MustBuild()

// Avoid (unless you need specific control)
config := core.Config{
    ErrorStrategy:  core.CollectAll,
    Timeout:        30 * time.Second,
    MaxConcurrency: 100,
    Context:        context.Background(),
}
```

### 2. Create Configuration Constants

```go
const (
    DefaultTimeout        = 30 * time.Second
    HighConcurrencyLimit  = 1000
    LowConcurrencyLimit   = 10
    MaxRetries           = 5
)

config := core.NewConfigBuilder().
    Timeout(DefaultTimeout).
    MaxConcurrency(HighConcurrencyLimit).
    Retries(MaxRetries).
    MustBuild()
```

### 3. Use Inheritance for Configuration Hierarchies

```go
// Organization defaults
orgDefaults := core.NewConfigBuilder().
    ErrorStrategy(core.FailFast).
    Timeout(30 * time.Second).
    MaxConcurrency(100).
    MustBuild()

// Service-specific overrides
serviceConfig := core.NewConfigBuilderFrom(orgDefaults).
    ErrorStrategy(core.CollectAll). // This service needs error collection
    MustBuild()
```

### 4. Validate Early and Often

```go
func CreateServiceConfig(timeout time.Duration, maxConcurrency int) (core.Config, error) {
    return core.NewConfigBuilder().
        Timeout(timeout).
        MaxConcurrency(maxConcurrency).
        ErrorStrategy(core.CollectAll).
        Build() // Validate immediately
}
```

### 5. Use Context for Cancellation

```go
func ProcessWithTimeout(ctx context.Context, timeout time.Duration) error {
    config := core.NewConfigBuilder().
        WithContext(ctx).
        Timeout(timeout).
        ErrorStrategy(core.CollectAll).
        MustBuild()
    
    // Use config for orchestration...
    return nil
}
```

## Common Patterns

### Configuration Factory

```go
type ConfigFactory struct {
    baseConfig core.Config
}

func NewConfigFactory() *ConfigFactory {
    return &ConfigFactory{
        baseConfig: core.DefaultConfig(),
    }
}

func (cf *ConfigFactory) ForBatchProcessing() core.Config {
    return core.NewConfigBuilderFrom(cf.baseConfig).
        ErrorStrategy(core.CollectAll).
        Timeout(5 * time.Minute).
        MaxConcurrency(50).
        MustBuild()
}

func (cf *ConfigFactory) ForRealTimeProcessing() core.Config {
    return core.NewConfigBuilderFrom(cf.baseConfig).
        ErrorStrategy(core.FailFast).
        Timeout(1 * time.Second).
        MaxConcurrency(1000).
        MustBuild()
}
```

### Environment-Based Configuration

```go
func ConfigFromEnvironment() core.Config {
    builder := core.NewConfigBuilder()
    
    if timeout := os.Getenv("ORCHESTRATOR_TIMEOUT"); timeout != "" {
        if d, err := time.ParseDuration(timeout); err == nil {
            builder.Timeout(d)
        }
    }
    
    if maxConcurrency := os.Getenv("ORCHESTRATOR_MAX_CONCURRENCY"); maxConcurrency != "" {
        if n, err := strconv.Atoi(maxConcurrency); err == nil {
            builder.MaxConcurrency(n)
        }
    }
    
    if strategy := os.Getenv("ORCHESTRATOR_ERROR_STRATEGY"); strategy == "collect_all" {
        builder.ErrorStrategy(core.CollectAll)
    }
    
    return builder.MustBuild()
}
```

### Testing Configuration

```go
func TestConfig() core.Config {
    return core.NewConfigBuilder().
        ErrorStrategy(core.CollectAll). // Collect all errors in tests
        Timeout(100 * time.Millisecond). // Fast timeouts for tests
        MaxConcurrency(10).              // Limited concurrency for tests
        Retries(0).                      // No retries in tests
        MustBuild()
}
```

## Migration Guide

### From Direct Config Creation

```go
// Old way
config := core.Config{
    ErrorStrategy:  core.CollectAll,
    Timeout:        30 * time.Second,
    MaxConcurrency: 100,
    Context:        context.Background(),
}

// New way
config := core.NewConfigBuilder().
    ErrorStrategy(core.CollectAll).
    Timeout(30 * time.Second).
    MaxConcurrency(100).
    MustBuild()
```

### From Manual Validation

```go
// Old way
config := core.Config{
    Timeout:        timeout,
    MaxConcurrency: maxConcurrency,
}
if timeout <= 0 {
    return fmt.Errorf("invalid timeout")
}
if maxConcurrency <= 0 {
    return fmt.Errorf("invalid maxConcurrency")
}

// New way
config, err := core.NewConfigBuilder().
    Timeout(timeout).
    MaxConcurrency(maxConcurrency).
    Build()
if err != nil {
    return err
}
```

## Performance Considerations

1. **Zero Allocations**: ConfigBuilder operations produce zero heap allocations
2. **Reuse Builders**: Clone builders instead of creating new ones
3. **Cache Configurations**: Reuse configurations when possible
4. **Validate Once**: Use `MustBuild()` when you're certain configuration is valid

## Troubleshooting

### Common Validation Errors

1. **Negative Timeout**: Ensure timeout is positive
2. **Zero MaxConcurrency**: Ensure MaxConcurrency is positive
3. **Negative Retries**: Ensure retries is non-negative
4. **Nil Context**: Always provide a valid context

### Performance Issues

1. **Too Many Allocations**: Use ConfigBuilder instead of direct Config creation
2. **Repeated Validation**: Cache validated configurations
3. **Large MaxConcurrency**: Consider system limits when setting MaxConcurrency

### Memory Leaks

1. **Context Leaks**: Ensure contexts are properly cancelled
2. **Builder Reuse**: Reset builders when reusing them