package core

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.ErrorStrategy != FailFast {
		t.Errorf("Expected ErrorStrategy to be FailFast, got %v", config.ErrorStrategy)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout to be 30s, got %v", config.Timeout)
	}

	if config.Retries != 0 {
		t.Errorf("Expected Retries to be 0, got %d", config.Retries)
	}

	if config.MaxConcurrency != 100 {
		t.Errorf("Expected MaxConcurrency to be 100, got %d", config.MaxConcurrency)
	}

	if config.Context != context.Background() {
		t.Error("Expected Context to be context.Background()")
	}
}

func TestConfigInherit(t *testing.T) {
	parent := Config{
		ErrorStrategy:  FailFast,
		Timeout:        10 * time.Second,
		Retries:        3,
		MaxConcurrency: 50,
		Context:        context.Background(),
	}

	child := Config{
		ErrorStrategy: CollectAll,
		Timeout:       5 * time.Second,
		// Retries and MaxConcurrency not set, should inherit from parent
	}

	result := child.Inherit(parent)

	if result.ErrorStrategy != CollectAll {
		t.Errorf("Expected ErrorStrategy to be CollectAll, got %v", result.ErrorStrategy)
	}

	if result.Timeout != 5*time.Second {
		t.Errorf("Expected Timeout to be 5s, got %v", result.Timeout)
	}

	if result.Retries != 3 {
		t.Errorf("Expected Retries to be inherited (3), got %d", result.Retries)
	}

	if result.MaxConcurrency != 50 {
		t.Errorf("Expected MaxConcurrency to be inherited (50), got %d", result.MaxConcurrency)
	}
}

func TestConfigInheritWithZeroValues(t *testing.T) {
	parent := DefaultConfig()
	child := Config{} // All zero values

	result := child.Inherit(parent)

	// Should inherit all values from parent since child has zero values
	if result.ErrorStrategy != parent.ErrorStrategy {
		t.Error("Should inherit ErrorStrategy from parent")
	}

	if result.Timeout != parent.Timeout {
		t.Error("Should inherit Timeout from parent")
	}

	if result.Retries != parent.Retries {
		t.Error("Should inherit Retries from parent")
	}

	if result.MaxConcurrency != parent.MaxConcurrency {
		t.Error("Should inherit MaxConcurrency from parent")
	}
}

func TestConfigHierarchicalInheritance(t *testing.T) {
	// Test complex inheritance scenarios
	grandparent := Config{
		ErrorStrategy:  FailFast,
		Timeout:        60 * time.Second,
		Retries:        3,
		MaxConcurrency: 200,
	}

	parent := Config{
		ErrorStrategy: CollectAll,
		Timeout:       30 * time.Second,
		// Retries and MaxConcurrency should inherit from grandparent
	}

	child := Config{
		Retries: 1,
		// Other fields should inherit from parent/grandparent chain
	}

	// First inheritance: parent inherits from grandparent
	parentResult := parent.Inherit(grandparent)

	if parentResult.ErrorStrategy != CollectAll {
		t.Error("Parent should override ErrorStrategy")
	}
	if parentResult.Timeout != 30*time.Second {
		t.Error("Parent should override Timeout")
	}
	if parentResult.Retries != 3 {
		t.Error("Parent should inherit Retries from grandparent")
	}
	if parentResult.MaxConcurrency != 200 {
		t.Error("Parent should inherit MaxConcurrency from grandparent")
	}

	// Second inheritance: child inherits from parent result
	childResult := child.Inherit(parentResult)

	if childResult.ErrorStrategy != CollectAll {
		t.Error("Child should inherit ErrorStrategy from parent")
	}
	if childResult.Timeout != 30*time.Second {
		t.Error("Child should inherit Timeout from parent")
	}
	if childResult.Retries != 1 {
		t.Error("Child should override Retries")
	}
	if childResult.MaxConcurrency != 200 {
		t.Error("Child should inherit MaxConcurrency from grandparent via parent")
	}
}

func TestConfigCompositionOverInheritance(t *testing.T) {
	// Test that we follow composition over inheritance principles
	base := DefaultConfig()

	// Create specialized configs by composing with base
	highConcurrencyConfig := Config{
		MaxConcurrency: 1000,
	}.Inherit(base)

	longRunningConfig := Config{
		Timeout: 300 * time.Second,
		Retries: 5,
	}.Inherit(base)

	errorTolerantConfig := Config{
		ErrorStrategy: CollectAll,
	}.Inherit(base)

	// Verify composition worked correctly
	if highConcurrencyConfig.MaxConcurrency != 1000 {
		t.Error("High concurrency config should have MaxConcurrency=1000")
	}
	if highConcurrencyConfig.ErrorStrategy != base.ErrorStrategy {
		t.Error("Should inherit ErrorStrategy from base")
	}

	if longRunningConfig.Timeout != 300*time.Second {
		t.Error("Long running config should have Timeout=300s")
	}
	if longRunningConfig.Retries != 5 {
		t.Error("Long running config should have Retries=5")
	}
	if longRunningConfig.MaxConcurrency != base.MaxConcurrency {
		t.Error("Should inherit MaxConcurrency from base")
	}

	if errorTolerantConfig.ErrorStrategy != CollectAll {
		t.Error("Error tolerant config should have ErrorStrategy=CollectAll")
	}
	if errorTolerantConfig.Timeout != base.Timeout {
		t.Error("Should inherit Timeout from base")
	}
}

// Benchmark tests for Config
func BenchmarkConfigInheritancePerformance(b *testing.B) {
	parent := NewConfigBuilder().
		ErrorStrategy(FailFast).
		Timeout(60 * time.Second).
		MaxConcurrency(200).
		MustBuild()

	child := NewConfigBuilder().
		Timeout(30 * time.Second).
		MustBuild()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		child.Inherit(parent)
	}
}
