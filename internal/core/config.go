// Package core provides the fundamental types and interfaces for the
// orchestrator library following Go best practices and KISS principles.
package core

import (
	"context"
	"time"
)

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
