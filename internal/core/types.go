// Package core provides the fundamental types and interfaces for the
// orchestrator library following Go best practices and KISS principles.
package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maniartech/orchestrator/internal/status"
)

// ErrorStrategy defines how errors should be handled during orchestration.
// It controls whether execution stops on the first error or continues to collect all errors.
//
// Example:
//
//	config := Config{ErrorStrategy: CollectAll}
//	// This will execute all operations even if some fail
type ErrorStrategy int

const (
	// FailFast stops execution immediately on first error.
	// Remaining operations are cancelled to minimize resource usage.
	FailFast ErrorStrategy = iota

	// CollectAll continues execution and collects all errors.
	// All operations run to completion regardless of individual failures.
	CollectAll
)

// String returns the string representation of ErrorStrategy
func (e ErrorStrategy) String() string {
	switch e {
	case FailFast:
		return "FailFast"
	case CollectAll:
		return "CollectAll"
	default:
		return "Unknown"
	}
}

// Config defines orchestration behavior with hierarchical inheritance.
// Child configurations inherit from parent configurations with local overrides.
// Zero values in child configs inherit the corresponding parent values.
//
// Example:
//
//	parentConfig := Config{
//	    ErrorStrategy: FailFast,
//	    Timeout: 30*time.Second,
//	    MaxConcurrency: 100,
//	}
//
//	childConfig := Config{
//	    Timeout: 10*time.Second, // Override parent timeout
//	    // ErrorStrategy and MaxConcurrency inherited from parent
//	}
//
//	finalConfig := childConfig.Inherit(parentConfig)
type Config struct {
	// ErrorStrategy controls how errors are handled (FailFast or CollectAll)
	ErrorStrategy ErrorStrategy

	// Timeout sets the maximum duration for operations
	Timeout time.Duration

	// Retries specifies the number of retry attempts for failed operations
	Retries int

	// MaxConcurrency limits the number of concurrent operations
	MaxConcurrency int

	// Context provides cancellation and deadline control
	Context context.Context
}

// DefaultConfig returns a config with sensible defaults.
// These defaults provide a good starting point for most orchestration scenarios.
//
// Default values:
//   - ErrorStrategy: FailFast (stop on first error)
//   - Timeout: 30 seconds
//   - Retries: 0 (no retries)
//   - MaxConcurrency: 100 operations
//   - Context: context.Background()
//
// Example:
//
//	config := DefaultConfig()
//	config.Timeout = 60*time.Second // Override timeout
func DefaultConfig() Config {
	return Config{
		ErrorStrategy:  FailFast,
		Timeout:        30 * time.Second,
		Retries:        0,
		MaxConcurrency: 100,
		Context:        context.Background(),
	}
}

// Inherit creates a new config that inherits from parent with local overrides.
// Child config values override parent values when non-zero.
// This enables hierarchical configuration where child orchestrations
// can override specific settings while inheriting others.
//
// Example:
//
//	parent := Config{
//	    ErrorStrategy: FailFast,
//	    Timeout: 30*time.Second,
//	    MaxConcurrency: 100,
//	}
//
//	child := Config{
//	    Timeout: 10*time.Second, // Override
//	    // ErrorStrategy and MaxConcurrency inherited
//	}
//
//	result := child.Inherit(parent)
//	// result.ErrorStrategy == FailFast (inherited)
//	// result.Timeout == 10*time.Second (overridden)
//	// result.MaxConcurrency == 100 (inherited)
func (c Config) Inherit(parent Config) Config {
	result := parent // Start with parent values

	// Override with non-zero values from child
	if c.ErrorStrategy != 0 || parent.ErrorStrategy == 0 {
		result.ErrorStrategy = c.ErrorStrategy
	}
	if c.Timeout != 0 {
		result.Timeout = c.Timeout
	}
	if c.Retries != 0 {
		result.Retries = c.Retries
	}
	if c.MaxConcurrency != 0 {
		result.MaxConcurrency = c.MaxConcurrency
	}
	if c.Context != nil {
		result.Context = c.Context
	}

	return result
}

// Context provides access to orchestration state and configuration.
// It enables inter-task communication and provides access to orchestration configuration.
//
// Example:
//
//	ctx.Set("user_id", 123)
//	if userID := ctx.Get("user_id"); userID != nil {
//	    fmt.Printf("Processing user: %v\n", userID)
//	}
type Context interface {
	// Get retrieves a value by name from the context
	Get(name string) any

	// Set stores a value by name in the context
	Set(name string, value any)

	// Config returns the current configuration
	Config() Config

	// Cancel signals cancellation to the orchestration
	Cancel()

	// Done returns a channel that's closed when cancellation occurs
	Done() <-chan struct{}
}

// Result contains named outputs and errors from orchestration execution.
// It provides thread-safe access to orchestration results with type safety.
//
// Example:
//
//	result := NewResult()
//	result.Set("user", User{ID: 123, Name: "John"})
//	if user, ok := result.GetTyped[User]("user"); ok {
//	    fmt.Printf("User: %+v\n", user)
//	}
type Result struct {
	entries map[string]any
	errors  []OperationError
	mu      sync.RWMutex
}

// NewResult creates a new result container with initialized maps and slices.
//
// Example:
//
//	result := NewResult()
//	result.Set("status", "completed")
func NewResult() *Result {
	return &Result{
		entries: make(map[string]any),
		errors:  make([]OperationError, 0),
	}
}

// Get retrieves a value by name from the result.
// Returns nil if the key doesn't exist.
//
// Example:
//
//	value := result.Get("status")
//	if value != nil {
//	    fmt.Printf("Status: %v\n", value)
//	}
func (r *Result) Get(name string) any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entries[name]
}

// GetTyped retrieves a value by name with type safety using generics.
// Returns the typed value and a boolean indicating if the value exists and matches the type.
//
// Example:
//
//	result.Set("count", 42)
//	if value, ok := GetTyped[int](result, "count"); ok {
//	    fmt.Printf("Count: %d\n", value)
//	}
func GetTyped[T any](r *Result, name string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var zero T
	value, exists := r.entries[name]
	if !exists {
		return zero, false
	}

	if typed, ok := value.(T); ok {
		return typed, true
	}

	return zero, false
}

// Set stores a value by name in the result.
// Thread-safe operation that can be called concurrently.
//
// Example:
//
//	result.Set("user_count", 42)
//	result.Set("processing_time", time.Since(start))
func (r *Result) Set(name string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[name] = value
}

// HasErrors returns true if there are any errors in the result
func (r *Result) HasErrors() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.errors) > 0
}

// Errors returns a copy of all errors
func (r *Result) Errors() []OperationError {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.errors) == 0 {
		return nil
	}

	// Return a copy to prevent external modification
	errors := make([]OperationError, len(r.errors))
	copy(errors, r.errors)
	return errors
}

// AddError adds an error to the result
func (r *Result) AddError(err OperationError) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, err)
}

// OperationError provides rich error information with metadata
type OperationError struct {
	Error     error
	Index     int
	Duration  time.Duration
	Timestamp time.Time
	OpID      string
	Stack     []byte
}

// Orchestration defines the interface for all orchestration types.
// It provides a fluent API for configuring orchestration behavior following the builder pattern.
// All orchestration types (Task, Sequential, Concurrent, Conditional) implement this interface.
//
// Example:
//
//	orchestration := someOrchestration.
//	    Named("user-processing").
//	    With(Config{Timeout: 30*time.Second}).
//	    ErrorBoundary(CollectAll)
type Orchestration interface {
	// Named sets a name for the orchestration for observability and debugging.
	// The name appears in logs and error messages to help identify which orchestration failed.
	Named(name string) Orchestration

	// With applies configuration to the orchestration.
	// Configuration is inherited hierarchically with local overrides.
	With(config Config) Orchestration

	// ErrorBoundary sets error handling strategy for this orchestration.
	// Controls how errors propagate within this orchestration scope.
	ErrorBoundary(strategy ErrorStrategy) Orchestration
}

// Executor defines the interface for executing orchestrations
type Executor interface {
	// Execute runs the orchestration and returns the result
	Execute(ctx context.Context, config Config) (*Result, error)
}

// OrchestrationFunc is a function type that implements Orchestration.
// It allows simple functions to be used as orchestrations.
//
// Example:
//
//	fn := OrchestrationFunc(func() (any, error) {
//	    return "Hello, World!", nil
//	})
type OrchestrationFunc func() (any, error)

// contextImpl provides a concrete implementation of Context
type contextImpl struct {
	values map[string]any
	config Config
	cancel context.CancelFunc
	done   <-chan struct{}
	mu     sync.RWMutex
}

// NewContext creates a new context implementation with the given configuration.
// The context inherits from DefaultConfig() and applies local overrides.
//
// Example:
//
//	config := Config{Timeout: 30*time.Second}
//	ctx := NewContext(config)
//	ctx.Set("start_time", time.Now())
func NewContext(config Config) Context {
	ctx, cancel := context.WithCancel(config.Context)

	return &contextImpl{
		values: make(map[string]any),
		config: config.Inherit(DefaultConfig()),
		cancel: cancel,
		done:   ctx.Done(),
	}
}

// Get retrieves a value by name from the context
func (c *contextImpl) Get(name string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.values[name]
}

// Set stores a value by name in the context
func (c *contextImpl) Set(name string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[name] = value
}

// Config returns the current configuration
func (c *contextImpl) Config() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// Cancel signals cancellation to the orchestration
func (c *contextImpl) Cancel() {
	c.cancel()
}

// Done returns a channel that's closed when cancellation occurs
func (c *contextImpl) Done() <-chan struct{} {
	return c.done
}

// ConfigBuilder provides fluent configuration construction using the builder pattern.
// It enables easy creation of configurations with method chaining and validation.
//
// Example:
//
//	config := NewConfigBuilder().
//	    ErrorStrategy(CollectAll).
//	    Timeout(30*time.Second).
//	    MaxConcurrency(200).
//	    Build()
type ConfigBuilder struct {
	config Config
	errors []error
}

// NewConfigBuilder creates a new ConfigBuilder with default values.
// The builder starts with DefaultConfig() as the base configuration.
//
// Example:
//
//	builder := NewConfigBuilder()
//	config := builder.Timeout(60*time.Second).Build()
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: DefaultConfig(),
		errors: make([]error, 0),
	}
}

// NewConfigBuilderFrom creates a new ConfigBuilder starting from an existing config.
// This enables creating variations of existing configurations.
//
// Example:
//
//	baseConfig := DefaultConfig()
//	builder := NewConfigBuilderFrom(baseConfig)
//	config := builder.MaxConcurrency(500).Build()
func NewConfigBuilderFrom(base Config) *ConfigBuilder {
	return &ConfigBuilder{
		config: base,
		errors: make([]error, 0),
	}
}

// ErrorStrategy sets the error handling strategy.
// Returns the builder for method chaining.
//
// Example:
//
//	config := NewConfigBuilder().
//	    ErrorStrategy(CollectAll).
//	    Build()
func (cb *ConfigBuilder) ErrorStrategy(strategy ErrorStrategy) *ConfigBuilder {
	if strategy != FailFast && strategy != CollectAll {
		cb.errors = append(cb.errors, fmt.Errorf("invalid ErrorStrategy: %d, must be FailFast (0) or CollectAll (1)", strategy))
		return cb
	}
	cb.config.ErrorStrategy = strategy
	return cb
}

// Timeout sets the maximum duration for operations.
// Returns the builder for method chaining.
// Validates that timeout is positive.
//
// Example:
//
//	config := NewConfigBuilder().
//	    Timeout(30*time.Second).
//	    Build()
func (cb *ConfigBuilder) Timeout(timeout time.Duration) *ConfigBuilder {
	if timeout <= 0 {
		cb.errors = append(cb.errors, fmt.Errorf("timeout must be positive, got %v", timeout))
		return cb
	}
	cb.config.Timeout = timeout
	return cb
}

// Retries sets the number of retry attempts for failed operations.
// Returns the builder for method chaining.
// Validates that retries is non-negative.
//
// Example:
//
//	config := NewConfigBuilder().
//	    Retries(3).
//	    Build()
func (cb *ConfigBuilder) Retries(retries int) *ConfigBuilder {
	if retries < 0 {
		cb.errors = append(cb.errors, fmt.Errorf("retries must be non-negative, got %d", retries))
		return cb
	}
	cb.config.Retries = retries
	return cb
}

// MaxConcurrency sets the maximum number of concurrent operations.
// Returns the builder for method chaining.
// Validates that maxConcurrency is positive.
//
// Example:
//
//	config := NewConfigBuilder().
//	    MaxConcurrency(100).
//	    Build()
func (cb *ConfigBuilder) MaxConcurrency(maxConcurrency int) *ConfigBuilder {
	if maxConcurrency <= 0 {
		cb.errors = append(cb.errors, fmt.Errorf("maxConcurrency must be positive, got %d", maxConcurrency))
		return cb
	}
	cb.config.MaxConcurrency = maxConcurrency
	return cb
}

// WithContext sets the context for cancellation and deadline control.
// Returns the builder for method chaining.
// Validates that context is not nil.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	config := NewConfigBuilder().
//	    WithContext(ctx).
//	    Build()
func (cb *ConfigBuilder) WithContext(ctx context.Context) *ConfigBuilder {
	if ctx == nil {
		cb.errors = append(cb.errors, fmt.Errorf("context cannot be nil"))
		return cb
	}
	cb.config.Context = ctx
	return cb
}

// Build constructs the final Config and validates all settings.
// Returns an error if any validation failed during the building process.
//
// Example:
//
//	config, err := NewConfigBuilder().
//	    ErrorStrategy(CollectAll).
//	    Timeout(30*time.Second).
//	    Build()
//	if err != nil {
//	    log.Fatal(err)
//	}
func (cb *ConfigBuilder) Build() (Config, error) {
	if len(cb.errors) > 0 {
		return Config{}, fmt.Errorf("configuration validation failed: %v", cb.errors)
	}

	// Perform final validation
	if err := cb.validate(); err != nil {
		return Config{}, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cb.config, nil
}

// MustBuild constructs the final Config and panics if validation fails.
// Use this only when you are certain the configuration is valid.
//
// Example:
//
//	config := NewConfigBuilder().
//	    ErrorStrategy(CollectAll).
//	    Timeout(30*time.Second).
//	    MustBuild()
func (cb *ConfigBuilder) MustBuild() Config {
	config, err := cb.Build()
	if err != nil {
		panic(fmt.Sprintf("ConfigBuilder.MustBuild() failed: %v", err))
	}
	return config
}

// validate performs comprehensive validation of the final configuration
func (cb *ConfigBuilder) validate() error {
	var validationErrors []error

	// Validate ErrorStrategy
	if cb.config.ErrorStrategy != FailFast && cb.config.ErrorStrategy != CollectAll {
		validationErrors = append(validationErrors, fmt.Errorf("invalid ErrorStrategy: %d", cb.config.ErrorStrategy))
	}

	// Validate Timeout
	if cb.config.Timeout <= 0 {
		validationErrors = append(validationErrors, fmt.Errorf("timeout must be positive: %v", cb.config.Timeout))
	}

	// Validate Retries
	if cb.config.Retries < 0 {
		validationErrors = append(validationErrors, fmt.Errorf("retries must be non-negative: %d", cb.config.Retries))
	}

	// Validate MaxConcurrency
	if cb.config.MaxConcurrency <= 0 {
		validationErrors = append(validationErrors, fmt.Errorf("maxConcurrency must be positive: %d", cb.config.MaxConcurrency))
	}

	// Validate Context
	if cb.config.Context == nil {
		validationErrors = append(validationErrors, fmt.Errorf("context cannot be nil"))
	}

	// Check for reasonable limits to prevent resource exhaustion
	if cb.config.MaxConcurrency > 10000 {
		validationErrors = append(validationErrors, fmt.Errorf("maxConcurrency too high (%d), maximum allowed is 10000", cb.config.MaxConcurrency))
	}

	if cb.config.Timeout > 24*time.Hour {
		validationErrors = append(validationErrors, fmt.Errorf("timeout too long (%v), maximum allowed is 24 hours", cb.config.Timeout))
	}

	if cb.config.Retries > 100 {
		validationErrors = append(validationErrors, fmt.Errorf("retries too high (%d), maximum allowed is 100", cb.config.Retries))
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("validation errors: %v", validationErrors)
	}

	return nil
}

// Reset clears all configuration and errors, returning to default state.
// This allows reusing the builder for multiple configurations.
//
// Example:
//
//	builder := NewConfigBuilder()
//	config1 := builder.Timeout(30*time.Second).MustBuild()
//	config2 := builder.Reset().Timeout(60*time.Second).MustBuild()
func (cb *ConfigBuilder) Reset() *ConfigBuilder {
	cb.config = DefaultConfig()
	cb.errors = cb.errors[:0] // Clear slice but keep capacity
	return cb
}

// Clone creates a copy of the current builder state.
// This allows creating variations without affecting the original builder.
//
// Example:
//
//	baseBuilder := NewConfigBuilder().ErrorStrategy(CollectAll)
//	fastBuilder := baseBuilder.Clone().Timeout(5*time.Second)
//	slowBuilder := baseBuilder.Clone().Timeout(60*time.Second)
func (cb *ConfigBuilder) Clone() *ConfigBuilder {
	clone := &ConfigBuilder{
		config: cb.config, // Config is a value type, so this creates a copy
		errors: make([]error, len(cb.errors)),
	}
	copy(clone.errors, cb.errors)
	return clone
}

// HasErrors returns true if there are any validation errors
func (cb *ConfigBuilder) HasErrors() bool {
	return len(cb.errors) > 0
}

// Errors returns a copy of all validation errors
func (cb *ConfigBuilder) Errors() []error {
	if len(cb.errors) == 0 {
		return nil
	}
	errors := make([]error, len(cb.errors))
	copy(errors, cb.errors)
	return errors
}

// StatusManager provides atomic status management for orchestrations
type StatusManager struct {
	*status.Manager
}

// NewStatusManager creates a new status manager
func NewStatusManager() *StatusManager {
	return &StatusManager{
		Manager: status.NewManager(),
	}
}
