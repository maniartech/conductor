package orchestration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/result"
)

// Test that our interface is properly defined
func TestOrchestrationInterface(t *testing.T) {
	// Test that our mock implementation satisfies the interface
	var _ Orchestration = (*mockOrchestration)(nil)
}

// mockOrchestration is a test implementation of Orchestration
type mockOrchestration struct {
	name          string
	config        *config.Config
	errorBoundary *errors.ErrorStrategy
	status        Status
}

func (m *mockOrchestration) Named(name string) Orchestration {
	m.name = name
	return m
}

func (m *mockOrchestration) With(config config.Config) Orchestration {
	m.config = &config
	return m
}

func (m *mockOrchestration) ErrorBoundary(strategy errors.ErrorStrategy) Orchestration {
	m.errorBoundary = &strategy
	return m
}

func (m *mockOrchestration) Execute(ctx context.Context, config config.Config) (*result.Result, error) {
	m.status = Running
	result := result.NewResult()
	result.Set("mock_result", "mock_value")
	m.status = Completed
	return result, nil
}

func (m *mockOrchestration) GetName() string {
	return m.name
}

func (m *mockOrchestration) GetConfig() *config.Config {
	return m.config
}

func (m *mockOrchestration) GetStatus() Status {
	return m.status
}

// PathResolver interface implementation for mockOrchestration
func (m *mockOrchestration) GetCurrentPath() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func (m *mockOrchestration) GetByPath(path string) (Orchestration, error) {
	if path == m.GetCurrentPath() {
		return m, nil
	}
	return nil, fmt.Errorf("path not found: %s", path)
}

func (m *mockOrchestration) ListAllPaths() []string {
	return []string{m.GetCurrentPath()}
}

func (m *mockOrchestration) FindByName(name string) []PathMatch {
	if m.name == name {
		return []PathMatch{
			{
				Path:          m.GetCurrentPath(),
				Orchestration: m,
				Depth:         0,
				Type:          "mock",
			},
		}
	}
	return []PathMatch{}
}

func (m *mockOrchestration) GetOrchestrationTree() *OrchestrationTree {
	return &OrchestrationTree{
		Name:          m.GetName(),
		Path:          m.GetCurrentPath(),
		Type:          "mock",
		Depth:         0,
		Orchestration: m,
		Children:      []*OrchestrationTree{},
	}
}

func (m *mockOrchestration) Query() *PathQuery {
	tree := m.GetOrchestrationTree()
	return NewPathQuery(tree)
}

func TestOrchestrationFluentAPI(t *testing.T) {
	mock := &mockOrchestration{}

	// Test fluent API chaining
	result := mock.
		Named("test-orchestration").
		With(config.Config{Timeout: 5 * time.Second}).
		ErrorBoundary(errors.CollectAll)

	// Verify the result is still the same instance (fluent API)
	if result.(*mockOrchestration) != mock {
		t.Error("Fluent API should return the same instance")
	}

	// Verify values were set
	if mock.name != "test-orchestration" {
		t.Errorf("Expected name 'test-orchestration', got %q", mock.name)
	}

	if mock.config == nil {
		t.Fatal("config.Config should not be nil")
	}

	if mock.config.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", mock.config.Timeout)
	}

	if mock.errorBoundary == nil {
		t.Fatal("ErrorBoundary should not be nil")
	}

	if *mock.errorBoundary != errors.CollectAll {
		t.Errorf("Expected errors.CollectAll, got %v", *mock.errorBoundary)
	}
}

func TestOrchestrationNamed(t *testing.T) {
	mock := &mockOrchestration{}

	// Test setting name
	result := mock.Named("my-orchestration")

	if result != mock {
		t.Error("Named should return the same instance")
	}

	if mock.name != "my-orchestration" {
		t.Errorf("Expected name 'my-orchestration', got %q", mock.name)
	}

	// Test overwriting name
	mock.Named("updated-name")
	if mock.name != "updated-name" {
		t.Errorf("Expected name 'updated-name', got %q", mock.name)
	}

	// Test empty name
	mock.Named("")
	if mock.name != "" {
		t.Errorf("Expected empty name, got %q", mock.name)
	}
}

func TestOrchestrationWith(t *testing.T) {
	mock := &mockOrchestration{}

	cfg := config.Config{
		ErrorStrategy:  errors.FailFast,
		Timeout:        30 * time.Second,
		Retries:        3,
		MaxConcurrency: 50,
	}

	// Test setting config
	result := mock.With(cfg)

	if result != mock {
		t.Error("With should return the same instance")
	}

	if mock.config == nil {
		t.Fatal("config.Config should not be nil")
	}

	if mock.config.ErrorStrategy != errors.FailFast {
		t.Error("errors.ErrorStrategy should be set")
	}

	if mock.config.Timeout != 30*time.Second {
		t.Error("Timeout should be set")
	}

	if mock.config.Retries != 3 {
		t.Error("Retries should be set")
	}

	if mock.config.MaxConcurrency != 50 {
		t.Error("MaxConcurrency should be set")
	}

	// Test overwriting config
	newConfig := config.Config{
		ErrorStrategy: errors.CollectAll,
		Timeout:       60 * time.Second,
	}

	mock.With(newConfig)

	if mock.config.ErrorStrategy != errors.CollectAll {
		t.Error("errors.ErrorStrategy should be updated")
	}

	if mock.config.Timeout != 60*time.Second {
		t.Error("Timeout should be updated")
	}
}

func TestOrchestrationErrorBoundary(t *testing.T) {
	mock := &mockOrchestration{}

	// Test setting error boundary
	result := mock.ErrorBoundary(errors.CollectAll)

	if result != mock {
		t.Error("ErrorBoundary should return the same instance")
	}

	if mock.errorBoundary == nil {
		t.Fatal("ErrorBoundary should not be nil")
	}

	if *mock.errorBoundary != errors.CollectAll {
		t.Errorf("Expected errors.CollectAll, got %v", *mock.errorBoundary)
	}

	// Test changing error boundary
	mock.ErrorBoundary(errors.FailFast)

	if *mock.errorBoundary != errors.FailFast {
		t.Errorf("Expected errors.FailFast, got %v", *mock.errorBoundary)
	}
}

func TestOrchestrationExecute(t *testing.T) {
	mock := &mockOrchestration{}

	ctx := context.Background()
	config := config.DefaultConfig()

	result, err := mock.Execute(ctx, config)

	if err != nil {
		t.Errorf("Execute should not return error, got: %v", err)
	}

	if result == nil {
		t.Fatal("result.Result should not be nil")
	}

	// Check mock result
	value := result.Get("mock_result")
	if value != "mock_value" {
		t.Errorf("Expected 'mock_value', got %v", value)
	}
}

func TestOrchestrationChainedExecution(t *testing.T) {
	mock := &mockOrchestration{}

	// Test full chain with execution
	ctx := context.Background()
	cfg := config.DefaultConfig()

	result, err := mock.
		Named("chained-execution").
		With(config.Config{Timeout: 10 * time.Second}).
		ErrorBoundary(errors.CollectAll).
		Execute(ctx, cfg)

	if err != nil {
		t.Errorf("Chained execution should not return error, got: %v", err)
	}

	if result == nil {
		t.Fatal("result.Result should not be nil")
	}

	// Verify configuration was applied
	if mock.name != "chained-execution" {
		t.Error("Name should be set from chain")
	}

	if mock.config == nil || mock.config.Timeout != 10*time.Second {
		t.Error("config.Config should be set from chain")
	}

	if mock.errorBoundary == nil || *mock.errorBoundary != errors.CollectAll {
		t.Error("ErrorBoundary should be set from chain")
	}
}

func TestOrchestrationConfigurationPropagation(t *testing.T) {
	mock := &mockOrchestration{}

	baseConfig := config.Config{
		ErrorStrategy:  errors.FailFast,
		Timeout:        30 * time.Second,
		MaxConcurrency: 100,
	}

	// Apply base configuration
	mock.With(baseConfig)

	// Override specific settings
	overrideConfig := config.Config{
		Timeout: 10 * time.Second,
		// Should inherit errors.ErrorStrategy and MaxConcurrency
	}

	mock.With(overrideConfig)

	// Verify the final configuration has the override
	if mock.config.Timeout != 10*time.Second {
		t.Error("Should have overridden timeout")
	}

	// Note: In a real implementation, the With method would need to handle inheritance
	// This test demonstrates the expected behavior pattern
}

func TestOrchestrationInterfaceCompliance(t *testing.T) {
	// Test that all methods return Orchestration interface
	mock := &mockOrchestration{}

	var orch Orchestration

	// Named should return Orchestration
	orch = mock.Named("test")
	if orch == nil {
		t.Error("Named should return Orchestration interface")
	}

	// With should return Orchestration
	orch = mock.With(config.DefaultConfig())
	if orch == nil {
		t.Error("With should return Orchestration interface")
	}

	// ErrorBoundary should return Orchestration
	orch = mock.ErrorBoundary(errors.FailFast)
	if orch == nil {
		t.Error("ErrorBoundary should return Orchestration interface")
	}

	// Execute should work with interface
	ctx := context.Background()
	config := config.DefaultConfig()
	result, err := orch.Execute(ctx, config)

	if err != nil {
		t.Errorf("Execute should work with interface, got error: %v", err)
	}

	if result == nil {
		t.Error("Execute should return result")
	}
}

func TestOrchestrationMethodOrder(t *testing.T) {
	// Test that methods can be called in any order
	mock1 := &mockOrchestration{}
	mock2 := &mockOrchestration{}
	mock3 := &mockOrchestration{}

	// Different orders should all work
	mock1.Named("test1").With(config.DefaultConfig()).ErrorBoundary(errors.FailFast)
	mock2.With(config.DefaultConfig()).ErrorBoundary(errors.CollectAll).Named("test2")
	mock3.ErrorBoundary(errors.FailFast).Named("test3").With(config.DefaultConfig())

	// All should have their values set correctly
	if mock1.name != "test1" {
		t.Error("mock1 name should be set")
	}

	if mock2.name != "test2" {
		t.Error("mock2 name should be set")
	}

	if mock3.name != "test3" {
		t.Error("mock3 name should be set")
	}

	// All should have configs
	if mock1.config == nil || mock2.config == nil || mock3.config == nil {
		t.Error("All mocks should have configs")
	}

	// All should have error boundaries
	if mock1.errorBoundary == nil || mock2.errorBoundary == nil || mock3.errorBoundary == nil {
		t.Error("All mocks should have error boundaries")
	}
}

// Example tests for documentation
func ExampleOrchestration() {
	mock := &mockOrchestration{}

	// Fluent API usage
	orchestration := mock.Named("user-processing").
		With(config.Config{Timeout: 30 * time.Second}).
		ErrorBoundary(errors.CollectAll)

	ctx := context.Background()
	config := config.DefaultConfig()

	result, err := orchestration.Execute(ctx, config)
	if err != nil {
		// Handle error
		return
	}

	// Use result
	_ = result.Get("mock_result")
}

// Benchmark tests
func BenchmarkOrchestrationFluentAPI(b *testing.B) {
	config := config.Config{
		ErrorStrategy:  errors.CollectAll,
		Timeout:        30 * time.Second,
		MaxConcurrency: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock := &mockOrchestration{}
		mock.Named("benchmark-test").
			With(config).
			ErrorBoundary(errors.FailFast)
	}
}

func BenchmarkOrchestrationExecution(b *testing.B) {
	mock := &mockOrchestration{}
	mock.Named("benchmark-execution")

	ctx := context.Background()
	config := config.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.Execute(ctx, config)
	}
}
