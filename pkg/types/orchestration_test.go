package types

import (
	"context"
	"fmt"
	"testing"

	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/result"
)

// MockOrchestration is a mock implementation of the Orchestration interface for testing
type MockOrchestration struct {
	name          string
	config        *config.Config
	errorBoundary *errors.ErrorStrategy
	status        Status
	executeFunc   func(ctx context.Context, config config.Config) (*result.Result, error)
}

// Named implements Orchestration interface
func (m *MockOrchestration) Named(name string) Orchestration {
	m.name = name
	return m
}

// With implements Orchestration interface
func (m *MockOrchestration) With(cfg config.Config) Orchestration {
	m.config = &cfg
	return m
}

// ErrorBoundary implements Orchestration interface
func (m *MockOrchestration) ErrorBoundary(strategy errors.ErrorStrategy) Orchestration {
	m.errorBoundary = &strategy
	return m
}

// Execute implements Orchestration interface
func (m *MockOrchestration) Execute(ctx context.Context, cfg config.Config) (*result.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cfg)
	}
	res := result.NewResult()
	res.Set(m.name, "mock-result")
	return res, nil
}

// GetName implements Orchestration interface
func (m *MockOrchestration) GetName() string {
	return m.name
}

// GetConfig implements Orchestration interface
func (m *MockOrchestration) GetConfig() *config.Config {
	return m.config
}

// GetStatus implements Orchestration interface
func (m *MockOrchestration) GetStatus() Status {
	return m.status
}

// PathResolver methods - minimal implementation for testing
func (m *MockOrchestration) GetByPath(path string) (Orchestration, error) {
	if path == m.name {
		return m, nil
	}
	return nil, fmt.Errorf("path not found: %s", path)
}

func (m *MockOrchestration) GetCurrentPath() string {
	return m.name
}

func (m *MockOrchestration) ListAllPaths() []string {
	return []string{m.name}
}

func (m *MockOrchestration) FindByName(name string) []PathMatch {
	if name == m.name {
		return []PathMatch{{
			Path:          m.name,
			Orchestration: m,
			Depth:         0,
			Type:          "mock",
		}}
	}
	return []PathMatch{}
}

func (m *MockOrchestration) GetOrchestrationTree() *OrchestrationTree {
	return &OrchestrationTree{
		Name:          m.name,
		Path:          m.name,
		Type:          "mock",
		Depth:         0,
		Orchestration: m,
	}
}

func (m *MockOrchestration) Query() *PathQuery {
	return NewPathQuery(m.GetOrchestrationTree())
}

// TestOrchestration_FluentAPI tests the fluent API pattern
func TestOrchestration_FluentAPI(t *testing.T) {
	mock := &MockOrchestration{}

	// Test method chaining
	result := mock.
		Named("test-orchestration").
		With(config.Config{Timeout: 30}).
		ErrorBoundary(errors.FailFast)

	if result != mock {
		t.Error("Expected fluent API to return the same instance")
	}

	if mock.GetName() != "test-orchestration" {
		t.Errorf("Expected name 'test-orchestration', got '%s'", mock.GetName())
	}

	if mock.config == nil {
		t.Fatal("Expected config to be set")
	}

	if mock.config.Timeout != 30 {
		t.Errorf("Expected timeout 30, got %v", mock.config.Timeout)
	}

	if mock.errorBoundary == nil {
		t.Fatal("Expected error boundary to be set")
	}

	if *mock.errorBoundary != errors.FailFast {
		t.Errorf("Expected FailFast strategy, got %v", *mock.errorBoundary)
	}
}

// TestOrchestration_Named tests the Named method
func TestOrchestration_Named(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple_name", "task-1", "task-1"},
		{"empty_name", "", ""},
		{"complex_name", "user-authentication-task", "user-authentication-task"},
		{"name_with_numbers", "step-123", "step-123"},
		{"name_with_special_chars", "auth-task_v2", "auth-task_v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockOrchestration{}
			result := mock.Named(tt.input)

			if result != mock {
				t.Error("Expected Named to return the same instance")
			}

			if mock.GetName() != tt.expected {
				t.Errorf("Expected name '%s', got '%s'", tt.expected, mock.GetName())
			}
		})
	}
}

// TestOrchestration_With tests the With method
func TestOrchestration_With(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
	}{
		{
			name: "timeout_config",
			config: config.Config{
				Timeout: 60,
			},
		},
		{
			name: "error_strategy_config",
			config: config.Config{
				ErrorStrategy: errors.CollectAll,
			},
		},
		{
			name: "max_concurrency_config",
			config: config.Config{
				MaxConcurrency: 10,
			},
		},
		{
			name: "full_config",
			config: config.Config{
				Timeout:        30,
				ErrorStrategy:  errors.FailFast,
				MaxConcurrency: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockOrchestration{}
			result := mock.With(tt.config)

			if result != mock {
				t.Error("Expected With to return the same instance")
			}

			if mock.config == nil {
				t.Fatal("Expected config to be set")
			}

			if mock.config.Timeout != tt.config.Timeout {
				t.Errorf("Expected timeout %v, got %v", tt.config.Timeout, mock.config.Timeout)
			}

			if mock.config.ErrorStrategy != tt.config.ErrorStrategy {
				t.Errorf("Expected error strategy %v, got %v", tt.config.ErrorStrategy, mock.config.ErrorStrategy)
			}

			if mock.config.MaxConcurrency != tt.config.MaxConcurrency {
				t.Errorf("Expected max concurrency %d, got %d", tt.config.MaxConcurrency, mock.config.MaxConcurrency)
			}
		})
	}
}

// TestOrchestration_ErrorBoundary tests the ErrorBoundary method
func TestOrchestration_ErrorBoundary(t *testing.T) {
	tests := []struct {
		name     string
		strategy errors.ErrorStrategy
	}{
		{"fail_fast", errors.FailFast},
		{"collect_all", errors.CollectAll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockOrchestration{}
			result := mock.ErrorBoundary(tt.strategy)

			if result != mock {
				t.Error("Expected ErrorBoundary to return the same instance")
			}

			if mock.errorBoundary == nil {
				t.Fatal("Expected error boundary to be set")
			}

			if *mock.errorBoundary != tt.strategy {
				t.Errorf("Expected strategy %v, got %v", tt.strategy, *mock.errorBoundary)
			}
		})
	}
}

// TestOrchestration_Execute tests the Execute method
func TestOrchestration_Execute(t *testing.T) {
	t.Run("successful_execution", func(t *testing.T) {
		mock := &MockOrchestration{
			name: "test-task",
		}

		ctx := context.Background()
		cfg := config.Config{}

		result, err := mock.Execute(ctx, cfg)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result to be returned")
		}

		value := result.Get("test-task")
		if value != "mock-result" {
			t.Errorf("Expected result 'mock-result', got %v", value)
		}
	})

	t.Run("execution_with_custom_function", func(t *testing.T) {
		expectedResult := result.NewResult()
		expectedResult.Set("custom", "custom-value")

		mock := &MockOrchestration{
			name: "custom-task",
			executeFunc: func(ctx context.Context, config config.Config) (*result.Result, error) {
				return expectedResult, nil
			},
		}

		ctx := context.Background()
		cfg := config.Config{}

		result, err := mock.Execute(ctx, cfg)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != expectedResult {
			t.Error("Expected custom result to be returned")
		}

		value := result.Get("custom")
		if value != "custom-value" {
			t.Errorf("Expected result 'custom-value', got %v", value)
		}
	})
}

// TestOrchestration_GetMethods tests getter methods
func TestOrchestration_GetMethods(t *testing.T) {
	cfg := config.Config{
		Timeout:        60,
		ErrorStrategy:  errors.CollectAll,
		MaxConcurrency: 10,
	}

	mock := &MockOrchestration{
		name:   "test-orchestration",
		config: &cfg,
		status: Running,
	}

	t.Run("get_name", func(t *testing.T) {
		if mock.GetName() != "test-orchestration" {
			t.Errorf("Expected name 'test-orchestration', got '%s'", mock.GetName())
		}
	})

	t.Run("get_config", func(t *testing.T) {
		config := mock.GetConfig()
		if config == nil {
			t.Fatal("Expected config to be returned")
		}

		if config.Timeout != 60 {
			t.Errorf("Expected timeout 60, got %v", config.Timeout)
		}

		if config.ErrorStrategy != errors.CollectAll {
			t.Errorf("Expected CollectAll strategy, got %v", config.ErrorStrategy)
		}

		if config.MaxConcurrency != 10 {
			t.Errorf("Expected max concurrency 10, got %d", config.MaxConcurrency)
		}
	})

	t.Run("get_status", func(t *testing.T) {
		if mock.GetStatus() != Running {
			t.Errorf("Expected status Running, got %v", mock.GetStatus())
		}
	})
}

// TestOrchestration_PathResolver tests PathResolver methods
func TestOrchestration_PathResolver(t *testing.T) {
	mock := &MockOrchestration{
		name: "test-orchestration",
	}

	t.Run("get_by_path", func(t *testing.T) {
		// Test finding self
		orch, err := mock.GetByPath("test-orchestration")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if orch != mock {
			t.Error("Expected to find self by path")
		}

		// Test path not found
		_, err = mock.GetByPath("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent path")
		}
	})

	t.Run("get_current_path", func(t *testing.T) {
		path := mock.GetCurrentPath()
		if path != "test-orchestration" {
			t.Errorf("Expected path 'test-orchestration', got '%s'", path)
		}
	})

	t.Run("list_all_paths", func(t *testing.T) {
		paths := mock.ListAllPaths()
		if len(paths) != 1 {
			t.Errorf("Expected 1 path, got %d", len(paths))
		}
		if paths[0] != "test-orchestration" {
			t.Errorf("Expected path 'test-orchestration', got '%s'", paths[0])
		}
	})

	t.Run("find_by_name", func(t *testing.T) {
		// Test finding self
		matches := mock.FindByName("test-orchestration")
		if len(matches) != 1 {
			t.Errorf("Expected 1 match, got %d", len(matches))
		}
		if matches[0].Orchestration != mock {
			t.Error("Expected to find self by name")
		}

		// Test name not found
		matches = mock.FindByName("non-existent")
		if len(matches) != 0 {
			t.Errorf("Expected 0 matches, got %d", len(matches))
		}
	})

	t.Run("get_orchestration_tree", func(t *testing.T) {
		tree := mock.GetOrchestrationTree()
		if tree == nil {
			t.Fatal("Expected tree to be returned")
		}
		if tree.Name != "test-orchestration" {
			t.Errorf("Expected tree name 'test-orchestration', got '%s'", tree.Name)
		}
		if tree.Orchestration != mock {
			t.Error("Expected tree to reference self")
		}
	})

	t.Run("query", func(t *testing.T) {
		query := mock.Query()
		if query == nil {
			t.Fatal("Expected query to be returned")
		}
		if query.tree.Name != "test-orchestration" {
			t.Errorf("Expected query tree name 'test-orchestration', got '%s'", query.tree.Name)
		}
	})
}

// TestOrchestration_InterfaceCompliance tests interface compliance
func TestOrchestration_InterfaceCompliance(t *testing.T) {
	var _ Orchestration = &MockOrchestration{}
	var _ Executor = &MockOrchestration{}
	var _ PathResolver = &MockOrchestration{}

	// This test ensures that MockOrchestration implements all required interfaces
	t.Log("✓ MockOrchestration implements all required interfaces")
}

// TestOrchestration_NilConfig tests behavior with nil config
func TestOrchestration_NilConfig(t *testing.T) {
	mock := &MockOrchestration{}

	config := mock.GetConfig()
	if config != nil {
		t.Error("Expected nil config initially")
	}

	// Test that methods work with nil config
	result := mock.Named("test").ErrorBoundary(errors.FailFast)
	if result != mock {
		t.Error("Expected methods to work with nil config")
	}
}

// TestOrchestration_ChainedOperations tests complex method chaining
func TestOrchestration_ChainedOperations(t *testing.T) {
	mock := &MockOrchestration{}

	// Test complex chaining
	result := mock.
		Named("complex-task").
		With(config.Config{Timeout: 30}).
		ErrorBoundary(errors.CollectAll).
		Named("renamed-task").                 // Override name
		With(config.Config{MaxConcurrency: 5}) // Override config

	if result != mock {
		t.Error("Expected chained operations to return the same instance")
	}

	if mock.GetName() != "renamed-task" {
		t.Errorf("Expected final name 'renamed-task', got '%s'", mock.GetName())
	}

	if mock.config.MaxConcurrency != 5 {
		t.Errorf("Expected final max concurrency 5, got %d", mock.config.MaxConcurrency)
	}

	if *mock.errorBoundary != errors.CollectAll {
		t.Errorf("Expected CollectAll strategy, got %v", *mock.errorBoundary)
	}
}

// BenchmarkOrchestration_Named benchmarks the Named method
func BenchmarkOrchestration_Named(b *testing.B) {
	mock := &MockOrchestration{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.Named("benchmark-task")
	}
}

// BenchmarkOrchestration_With benchmarks the With method
func BenchmarkOrchestration_With(b *testing.B) {
	mock := &MockOrchestration{}
	cfg := config.Config{Timeout: 30}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.With(cfg)
	}
}

// BenchmarkOrchestration_Execute benchmarks the Execute method
func BenchmarkOrchestration_Execute(b *testing.B) {
	mock := &MockOrchestration{name: "benchmark-task"}
	ctx := context.Background()
	cfg := config.Config{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.Execute(ctx, cfg)
	}
}
