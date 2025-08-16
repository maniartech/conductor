package config

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/errors"
)

func TestNewConfigBuilder(t *testing.T) {
	builder := NewConfigBuilder()

	if builder == nil {
		t.Fatal("NewConfigBuilder() returned nil")
	}

	if builder.config.ErrorStrategy != errors.FailFast {
		t.Error("Should start with default ErrorStrategy")
	}

	if builder.config.Timeout != 30*time.Second {
		t.Error("Should start with default Timeout")
	}

	if builder.config.MaxConcurrency != 100 {
		t.Error("Should start with default MaxConcurrency")
	}

	if len(builder.errors) != 0 {
		t.Error("Should start with no errors")
	}
}

func TestNewConfigBuilderFrom(t *testing.T) {
	base := Config{
		ErrorStrategy:  errors.CollectAll,
		Timeout:        60 * time.Second,
		Retries:        5,
		MaxConcurrency: 200,
		Context:        context.Background(),
	}

	builder := NewConfigBuilderFrom(base)

	if builder.config.ErrorStrategy != errors.CollectAll {
		t.Error("Should inherit ErrorStrategy from base")
	}

	if builder.config.Timeout != 60*time.Second {
		t.Error("Should inherit Timeout from base")
	}

	if builder.config.Retries != 5 {
		t.Error("Should inherit Retries from base")
	}

	if builder.config.MaxConcurrency != 200 {
		t.Error("Should inherit MaxConcurrency from base")
	}
}

func TestConfigBuilderFluentAPI(t *testing.T) {
	builder := NewConfigBuilder()

	// Test method chaining
	result := builder.
		ErrorStrategy(errors.CollectAll).
		Timeout(45 * time.Second).
		Retries(3).
		MaxConcurrency(150)

	// Should return the same builder instance
	if result != builder {
		t.Error("Fluent API should return the same builder instance")
	}

	// Verify values were set
	if builder.config.ErrorStrategy != errors.CollectAll {
		t.Error("ErrorStrategy should be set")
	}

	if builder.config.Timeout != 45*time.Second {
		t.Error("Timeout should be set")
	}

	if builder.config.Retries != 3 {
		t.Error("Retries should be set")
	}

	if builder.config.MaxConcurrency != 150 {
		t.Error("MaxConcurrency should be set")
	}
}

func TestConfigBuilderValidation(t *testing.T) {
	tests := []struct {
		name          string
		builderFunc   func(*ConfigBuilder) *ConfigBuilder
		expectError   bool
		errorContains string
	}{
		{
			name: "valid configuration",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.ErrorStrategy(errors.CollectAll).Timeout(30 * time.Second).MaxConcurrency(100)
			},
			expectError: false,
		},
		{
			name: "invalid error strategy",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.ErrorStrategy(errors.ErrorStrategy(999))
			},
			expectError:   true,
			errorContains: "invalid ErrorStrategy",
		},
		{
			name: "negative timeout",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.Timeout(-1 * time.Second)
			},
			expectError:   true,
			errorContains: "timeout must be positive",
		},
		{
			name: "zero timeout",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.Timeout(0)
			},
			expectError:   true,
			errorContains: "timeout must be positive",
		},
		{
			name: "negative retries",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.Retries(-1)
			},
			expectError:   true,
			errorContains: "retries must be non-negative",
		},
		{
			name: "zero max concurrency",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.MaxConcurrency(0)
			},
			expectError:   true,
			errorContains: "maxConcurrency must be positive",
		},
		{
			name: "negative max concurrency",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.MaxConcurrency(-1)
			},
			expectError:   true,
			errorContains: "maxConcurrency must be positive",
		},
		{
			name: "nil context",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.WithContext(nil)
			},
			expectError:   true,
			errorContains: "context cannot be nil",
		},
		{
			name: "max concurrency too high",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.MaxConcurrency(20000)
			},
			expectError:   true,
			errorContains: "maxConcurrency too high",
		},
		{
			name: "timeout too long",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.Timeout(25 * time.Hour)
			},
			expectError:   true,
			errorContains: "timeout too long",
		},
		{
			name: "retries too high",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.Retries(150)
			},
			expectError:   true,
			errorContains: "retries too high",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := NewConfigBuilder()
			builder = test.builderFunc(builder)

			config, err := builder.Build()

			if test.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !contains(err.Error(), test.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", test.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if config.Context == nil {
					t.Error("Valid config should have non-nil context")
				}
			}
		})
	}
}

func TestConfigBuilderMustBuild(t *testing.T) {
	// Test successful MustBuild
	builder := NewConfigBuilder()
	config := builder.ErrorStrategy(errors.CollectAll).MustBuild()

	if config.ErrorStrategy != errors.CollectAll {
		t.Error("MustBuild should return valid config")
	}

	// Test MustBuild panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild should panic on invalid config")
		}
	}()

	invalidBuilder := NewConfigBuilder()
	invalidBuilder.Timeout(-1 * time.Second).MustBuild()
}

func TestConfigBuilderReset(t *testing.T) {
	builder := NewConfigBuilder()

	// Modify the builder
	builder.ErrorStrategy(errors.CollectAll).Timeout(60 * time.Second).Retries(5)

	// Add an error
	builder.MaxConcurrency(-1)

	// Reset should restore defaults
	builder.Reset()

	if builder.config.ErrorStrategy != errors.FailFast {
		t.Error("Reset should restore default ErrorStrategy")
	}

	if builder.config.Timeout != 30*time.Second {
		t.Error("Reset should restore default Timeout")
	}

	if builder.config.Retries != 0 {
		t.Error("Reset should restore default Retries")
	}

	if len(builder.errors) != 0 {
		t.Error("Reset should clear errors")
	}
}

func TestConfigBuilderClone(t *testing.T) {
	original := NewConfigBuilder()
	original.ErrorStrategy(errors.CollectAll).Timeout(60 * time.Second)

	clone := original.Clone()

	// Should be different instances
	if clone == original {
		t.Error("Clone should return different instance")
	}

	// Should have same configuration
	if clone.config.ErrorStrategy != original.config.ErrorStrategy {
		t.Error("Clone should have same ErrorStrategy")
	}

	if clone.config.Timeout != original.config.Timeout {
		t.Error("Clone should have same Timeout")
	}

	// Modifying clone should not affect original
	clone.MaxConcurrency(500)

	if original.config.MaxConcurrency == 500 {
		t.Error("Modifying clone should not affect original")
	}
}

// Benchmark tests for ConfigBuilder
func BenchmarkConfigBuilderBuild(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewConfigBuilder().
			ErrorStrategy(errors.CollectAll).
			Timeout(30 * time.Second).
			Retries(3).
			MaxConcurrency(100).
			MustBuild()
	}
}

func BenchmarkConfigBuilderClone(b *testing.B) {
	builder := NewConfigBuilder().
		ErrorStrategy(errors.CollectAll).
		Timeout(30 * time.Second).
		Retries(3).
		MaxConcurrency(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.Clone()
	}
}

// Example tests for documentation
func ExampleNewConfigBuilder() {
	config := NewConfigBuilder().
		ErrorStrategy(errors.CollectAll).
		Timeout(30 * time.Second).
		MaxConcurrency(200).
		MustBuild()

	fmt.Printf("ErrorStrategy: %v\n", config.ErrorStrategy)
	fmt.Printf("Timeout: %v\n", config.Timeout)
	fmt.Printf("MaxConcurrency: %d\n", config.MaxConcurrency)

	// Output:
	// ErrorStrategy: CollectAll
	// Timeout: 30s
	// MaxConcurrency: 200
}

func ExampleConfigBuilder_Build() {
	config, err := NewConfigBuilder().
		ErrorStrategy(errors.CollectAll).
		Timeout(30 * time.Second).
		Build()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Config built successfully: %v\n", config.ErrorStrategy)

	// Output:
	// Config built successfully: CollectAll
}

// Helper function for string contains check (same as in task_builder_test.go)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
