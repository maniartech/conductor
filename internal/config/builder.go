// Package config provides configuration management for the orchestrator library.
// It includes ConfigBuilder for fluent configuration construction using the builder pattern.
package config

import (
	"context"
	"fmt"
	"time"

	"github.com/maniartech/orchestrator/internal/errors"
)

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
func (cb *ConfigBuilder) ErrorStrategy(strategy errors.ErrorStrategy) *ConfigBuilder {
	if strategy != errors.FailFast && strategy != errors.CollectAll {
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
	if cb.config.ErrorStrategy != errors.FailFast && cb.config.ErrorStrategy != errors.CollectAll {
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
