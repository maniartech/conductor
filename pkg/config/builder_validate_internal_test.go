package config

import (
	"strings"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/errors"
)

// Directly exercise validate() branches by constructing builders with invalid
// internal state but no pre-populated cb.errors so Build() reaches validate.
func TestConfigBuilderValidateMultipleErrorsCore(t *testing.T) {
	b := &ConfigBuilder{ // all core invalid conditions
		config: Config{
			ErrorStrategy:  errors.ErrorStrategy(42), // invalid
			Timeout:        -5 * time.Second,         // non-positive
			Retries:        -2,                       // negative
			MaxConcurrency: 0,                        // non-positive
			Context:        nil,                      // nil context
		},
		errors: make([]error, 0),
	}

	_, err := b.Build()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	msg := err.Error()
	// Check presence of each specific validation branch substring
	expected := []string{
		"invalid ErrorStrategy",
		"timeout must be positive",
		"retries must be non-negative",
		"maxConcurrency must be positive",
		"context cannot be nil",
	}
	for _, e := range expected {
		if !strings.Contains(msg, e) {
			t.Fatalf("expected error message to contain %q, got: %s", e, msg)
		}
	}
}

func TestConfigBuilderValidateReasonableLimits(t *testing.T) {
	b := &ConfigBuilder{ // exceed high-limit validations only
		config: Config{
			ErrorStrategy:  errors.FailFast,         // valid
			Timeout:        25 * time.Hour,          // >24h
			Retries:        150,                     // >100
			MaxConcurrency: 20001,                   // >10000
			Context:        DefaultConfig().Context, // non-nil
		},
		errors: make([]error, 0),
	}

	_, err := b.Build()
	if err == nil {
		t.Fatalf("expected validation error for limit breaches")
	}
	msg := err.Error()
	expected := []string{
		"timeout too long",
		"retries too high",
		"maxConcurrency too high",
	}
	for _, e := range expected {
		if !strings.Contains(msg, e) {
			t.Fatalf("expected error message to contain %q, got: %s", e, msg)
		}
	}
}
