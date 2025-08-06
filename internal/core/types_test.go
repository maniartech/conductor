package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestErrorStrategyString(t *testing.T) {
	tests := []struct {
		strategy ErrorStrategy
		expected string
	}{
		{FailFast, "FailFast"},
		{CollectAll, "CollectAll"},
		{ErrorStrategy(999), "Unknown"},
	}

	for _, test := range tests {
		if got := test.strategy.String(); got != test.expected {
			t.Errorf("ErrorStrategy(%d).String() = %q, want %q", test.strategy, got, test.expected)
		}
	}
}

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

func TestNewResult(t *testing.T) {
	result := NewResult()

	if result == nil {
		t.Fatal("NewResult() returned nil")
	}

	if result.entries == nil {
		t.Error("entries map should be initialized")
	}

	if result.errors == nil {
		t.Error("errors slice should be initialized")
	}

	if len(result.entries) != 0 {
		t.Error("entries map should be empty")
	}

	if len(result.errors) != 0 {
		t.Error("errors slice should be empty")
	}
}

func TestResultSetAndGet(t *testing.T) {
	result := NewResult()

	// Test setting and getting values
	result.Set("string", "test")
	result.Set("int", 42)
	result.Set("bool", true)

	if got := result.Get("string"); got != "test" {
		t.Errorf("Expected 'test', got %v", got)
	}

	if got := result.Get("int"); got != 42 {
		t.Errorf("Expected 42, got %v", got)
	}

	if got := result.Get("bool"); got != true {
		t.Errorf("Expected true, got %v", got)
	}

	if got := result.Get("nonexistent"); got != nil {
		t.Errorf("Expected nil for nonexistent key, got %v", got)
	}
}

func TestResultGetTyped(t *testing.T) {
	result := NewResult()

	result.Set("string", "test")
	result.Set("int", 42)
	result.Set("bool", true)

	// Test successful type assertions
	if str, ok := GetTyped[string](result, "string"); !ok {
		t.Error("GetTyped should have succeeded for string")
	} else if str != "test" {
		t.Errorf("Expected 'test', got %q", str)
	}

	if num, ok := GetTyped[int](result, "int"); !ok {
		t.Error("GetTyped should have succeeded for int")
	} else if num != 42 {
		t.Errorf("Expected 42, got %d", num)
	}

	if flag, ok := GetTyped[bool](result, "bool"); !ok {
		t.Error("GetTyped should have succeeded for bool")
	} else if !flag {
		t.Error("Expected true, got false")
	}

	// Test any type
	if iface, ok := GetTyped[any](result, "string"); !ok {
		t.Error("GetTyped should have succeeded for any")
	} else if iface != "test" {
		t.Errorf("Expected 'test', got %v", iface)
	}

	// Test failed type assertion
	if wrongType, ok := GetTyped[int](result, "string"); ok {
		t.Errorf("GetTyped should have failed for wrong type, got %d", wrongType)
	}

	// Test nonexistent key
	if missing, ok := GetTyped[string](result, "nonexistent"); ok {
		t.Errorf("GetTyped should have failed for nonexistent key, got %q", missing)
	}
}

func TestResultErrors(t *testing.T) {
	result := NewResult()

	// Initially no errors
	if result.HasErrors() {
		t.Error("HasErrors should return false initially")
	}

	if errors := result.Errors(); errors != nil {
		t.Error("Errors should return nil when no errors")
	}

	// Add an error
	opErr := OperationError{
		Error:     errors.New("test error"),
		Index:     0,
		Duration:  time.Millisecond,
		Timestamp: time.Now(),
		OpID:      "test-op",
	}

	result.AddError(opErr)

	if !result.HasErrors() {
		t.Error("HasErrors should return true after adding error")
	}

	resultErrors := result.Errors()
	if len(resultErrors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(resultErrors))
	}

	if resultErrors[0].Error.Error() != "test error" {
		t.Errorf("Expected 'test error', got %q", resultErrors[0].Error.Error())
	}

	// Verify it's a copy (modifying returned slice shouldn't affect original)
	resultErrors[0].Index = 999
	newErrors := result.Errors()
	if newErrors[0].Index == 999 {
		t.Error("Errors() should return a copy, not the original slice")
	}
}

func TestResultConcurrentAccess(t *testing.T) {
	result := NewResult()
	const numGoroutines = 100
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Readers and writers

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				result.Set(key, j)

				if j%10 == 0 {
					opErr := OperationError{
						Error: errors.New("test error"),
						Index: j,
					}
					result.AddError(opErr)
				}
			}
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				result.Get(key)
				result.HasErrors()
				result.Errors()
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	if !result.HasErrors() {
		t.Error("Should have errors after concurrent operations")
	}
}

func TestNewContext(t *testing.T) {
	config := DefaultConfig()
	ctx := NewContext(config)

	if ctx == nil {
		t.Fatal("NewContext() returned nil")
	}

	if ctx.Config().ErrorStrategy != config.ErrorStrategy {
		t.Error("Context should inherit config")
	}

	if ctx.Done() == nil {
		t.Error("Done() should return a channel")
	}
}

func TestContextSetAndGet(t *testing.T) {
	config := DefaultConfig()
	ctx := NewContext(config)

	// Test setting and getting values
	ctx.Set("test", "value")
	ctx.Set("number", 42)

	if got := ctx.Get("test"); got != "value" {
		t.Errorf("Expected 'value', got %v", got)
	}

	if got := ctx.Get("number"); got != 42 {
		t.Errorf("Expected 42, got %v", got)
	}

	if got := ctx.Get("nonexistent"); got != nil {
		t.Errorf("Expected nil for nonexistent key, got %v", got)
	}
}

func TestContextCancel(t *testing.T) {
	config := DefaultConfig()
	ctx := NewContext(config)

	done := ctx.Done()

	// Should not be cancelled initially
	select {
	case <-done:
		t.Error("Context should not be cancelled initially")
	default:
		// Expected
	}

	// Cancel the context
	ctx.Cancel()

	// Should be cancelled now
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after Cancel()")
	}
}

func TestContextConcurrentAccess(t *testing.T) {
	config := DefaultConfig()
	ctx := NewContext(config)

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			ctx.Set(key, id)
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			ctx.Get(key)
			ctx.Config()
		}(i)
	}

	wg.Wait()
}

func TestNewStatusManager(t *testing.T) {
	sm := NewStatusManager()

	if sm == nil {
		t.Fatal("NewStatusManager() returned nil")
	}

	if sm.Manager == nil {
		t.Error("StatusManager.Manager should not be nil")
	}
}

// Benchmark tests
func BenchmarkResultSet(b *testing.B) {
	result := NewResult()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.Set("key", i)
	}
}

func BenchmarkResultGet(b *testing.B) {
	result := NewResult()
	result.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.Get("key")
	}
}

func BenchmarkResultGetTyped(b *testing.B) {
	result := NewResult()
	result.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTyped[string](result, "key")
	}
}

func BenchmarkContextSet(b *testing.B) {
	config := DefaultConfig()
	ctx := NewContext(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Set("key", i)
	}
}

func BenchmarkContextGet(b *testing.B) {
	config := DefaultConfig()
	ctx := NewContext(config)
	ctx.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Get("key")
	}
}

func BenchmarkConcurrentResultAccess(b *testing.B) {
	result := NewResult()
	result.Set("key", "value")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result.Get("key")
		}
	})
}

func BenchmarkConcurrentContextAccess(b *testing.B) {
	config := DefaultConfig()
	ctx := NewContext(config)
	ctx.Set("key", "value")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx.Get("key")
		}
	})
}

// Interface compliance tests
func TestOrchestrationInterface(t *testing.T) {
	// Test that our types implement the Orchestration interface
	var _ Orchestration = (*mockOrchestration)(nil)
}

// mockOrchestration is a test implementation of Orchestration
type mockOrchestration struct {
	name          string
	config        *Config
	errorBoundary *ErrorStrategy
}

func (m *mockOrchestration) Named(name string) Orchestration {
	m.name = name
	return m
}

func (m *mockOrchestration) With(config Config) Orchestration {
	m.config = &config
	return m
}

func (m *mockOrchestration) ErrorBoundary(strategy ErrorStrategy) Orchestration {
	m.errorBoundary = &strategy
	return m
}

func TestOrchestrationFluentAPI(t *testing.T) {
	mock := &mockOrchestration{}

	// Test fluent API chaining
	result := mock.
		Named("test-orchestration").
		With(Config{Timeout: 5 * time.Second}).
		ErrorBoundary(CollectAll)

	// Verify the result is still the same instance (fluent API)
	if result != mock {
		t.Error("Fluent API should return the same instance")
	}

	// Verify values were set
	if mock.name != "test-orchestration" {
		t.Errorf("Expected name 'test-orchestration', got %q", mock.name)
	}

	if mock.config == nil {
		t.Fatal("Config should not be nil")
	}

	if mock.config.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", mock.config.Timeout)
	}

	if mock.errorBoundary == nil {
		t.Fatal("ErrorBoundary should not be nil")
	}

	if *mock.errorBoundary != CollectAll {
		t.Errorf("Expected CollectAll, got %v", *mock.errorBoundary)
	}
}

func TestContextInterface(t *testing.T) {
	// Test that contextImpl implements Context interface
	var _ Context = (*contextImpl)(nil)

	config := DefaultConfig()
	ctx := NewContext(config)

	// Test interface methods
	ctx.Set("test", "value")
	if got := ctx.Get("test"); got != "value" {
		t.Errorf("Expected 'value', got %v", got)
	}

	if cfg := ctx.Config(); cfg.ErrorStrategy != config.ErrorStrategy {
		t.Error("Config should match")
	}

	// Test cancellation
	done := ctx.Done()
	select {
	case <-done:
		t.Error("Should not be cancelled initially")
	default:
		// Expected
	}

	ctx.Cancel()

	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Should be cancelled after Cancel()")
	}
}

// Type safety tests
func TestResultTypeSafety(t *testing.T) {
	result := NewResult()

	// Test with custom types
	type User struct {
		ID   int
		Name string
	}

	user := User{ID: 123, Name: "John Doe"}
	result.Set("user", user)

	// Test successful type retrieval
	if retrievedUser, ok := GetTyped[User](result, "user"); !ok {
		t.Error("Should retrieve User type successfully")
	} else {
		if retrievedUser.ID != 123 {
			t.Errorf("Expected ID 123, got %d", retrievedUser.ID)
		}
		if retrievedUser.Name != "John Doe" {
			t.Errorf("Expected name 'John Doe', got %q", retrievedUser.Name)
		}
	}

	// Test type mismatch
	if _, ok := GetTyped[string](result, "user"); ok {
		t.Error("Should fail when requesting wrong type")
	}

	// Test with interface types
	result.Set("interface", any("test"))
	if value, ok := GetTyped[any](result, "interface"); !ok {
		t.Error("Should retrieve any type successfully")
	} else if value != "test" {
		t.Errorf("Expected 'test', got %v", value)
	}

	// Test with pointer types
	intPtr := new(int)
	*intPtr = 42
	result.Set("pointer", intPtr)

	if retrievedPtr, ok := GetTyped[*int](result, "pointer"); !ok {
		t.Error("Should retrieve pointer type successfully")
	} else if *retrievedPtr != 42 {
		t.Errorf("Expected 42, got %d", *retrievedPtr)
	}
}

func TestResultGenericTypeSafety(t *testing.T) {
	result := NewResult()

	// Test with slice types
	slice := []string{"a", "b", "c"}
	result.Set("slice", slice)

	if retrievedSlice, ok := GetTyped[[]string](result, "slice"); !ok {
		t.Error("Should retrieve slice type successfully")
	} else {
		if len(retrievedSlice) != 3 {
			t.Errorf("Expected length 3, got %d", len(retrievedSlice))
		}
		if retrievedSlice[0] != "a" {
			t.Errorf("Expected 'a', got %q", retrievedSlice[0])
		}
	}

	// Test with map types
	m := map[string]int{"one": 1, "two": 2}
	result.Set("map", m)

	if retrievedMap, ok := GetTyped[map[string]int](result, "map"); !ok {
		t.Error("Should retrieve map type successfully")
	} else {
		if retrievedMap["one"] != 1 {
			t.Errorf("Expected 1, got %d", retrievedMap["one"])
		}
		if retrievedMap["two"] != 2 {
			t.Errorf("Expected 2, got %d", retrievedMap["two"])
		}
	}

	// Test with channel types
	ch := make(chan int, 1)
	ch <- 42
	result.Set("channel", ch)

	if retrievedCh, ok := GetTyped[chan int](result, "channel"); !ok {
		t.Error("Should retrieve channel type successfully")
	} else {
		select {
		case value := <-retrievedCh:
			if value != 42 {
				t.Errorf("Expected 42, got %d", value)
			}
		default:
			t.Error("Channel should have a value")
		}
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

// Integration tests for interface interactions
func TestInterfaceInteractions(t *testing.T) {
	// Test interaction between Context and Result
	config := DefaultConfig()
	ctx := NewContext(config)
	result := NewResult()

	// Context sets values that Result can access
	ctx.Set("processing_start", time.Now())
	ctx.Set("user_id", 123)

	// Simulate transferring context values to result
	if startTime := ctx.Get("processing_start"); startTime != nil {
		result.Set("start_time", startTime)
	}

	if userID := ctx.Get("user_id"); userID != nil {
		result.Set("user_id", userID)
	}

	// Verify type-safe retrieval from result
	if userID, ok := GetTyped[int](result, "user_id"); !ok {
		t.Error("Should retrieve user_id as int")
	} else if userID != 123 {
		t.Errorf("Expected user_id 123, got %d", userID)
	}

	if startTime, ok := GetTyped[time.Time](result, "start_time"); !ok {
		t.Error("Should retrieve start_time as time.Time")
	} else if startTime.IsZero() {
		t.Error("Start time should not be zero")
	}
}

func TestOrchestrationConfigurationPropagation(t *testing.T) {
	// Test that configuration propagates correctly through orchestration hierarchy
	mock := &mockOrchestration{}

	baseConfig := Config{
		ErrorStrategy:  FailFast,
		Timeout:        30 * time.Second,
		MaxConcurrency: 100,
	}

	// Apply base configuration
	mock.With(baseConfig)

	// Override specific settings
	overrideConfig := Config{
		Timeout: 10 * time.Second,
		// Should inherit ErrorStrategy and MaxConcurrency
	}

	mock.With(overrideConfig)

	// Verify the final configuration has the override
	if mock.config.Timeout != 10*time.Second {
		t.Error("Should have overridden timeout")
	}

	// Note: In a real implementation, the With method would need to handle inheritance
	// This test demonstrates the expected behavior
}

// Documentation example tests (these serve as executable documentation)
func ExampleGetTyped() {
	result := NewResult()
	result.Set("count", 42)
	result.Set("message", "Hello, World!")

	// Type-safe retrieval
	if count, ok := GetTyped[int](result, "count"); ok {
		fmt.Printf("Count: %d\n", count)
	}

	if message, ok := GetTyped[string](result, "message"); ok {
		fmt.Printf("Message: %s\n", message)
	}

	// Output:
	// Count: 42
	// Message: Hello, World!
}

func ExampleConfig_Inherit() {
	parent := Config{
		ErrorStrategy:  FailFast,
		Timeout:        30 * time.Second,
		MaxConcurrency: 100,
	}

	child := Config{
		Timeout: 10 * time.Second,
		// ErrorStrategy and MaxConcurrency will be inherited
	}

	result := child.Inherit(parent)

	fmt.Printf("ErrorStrategy: %v\n", result.ErrorStrategy)
	fmt.Printf("Timeout: %v\n", result.Timeout)
	fmt.Printf("MaxConcurrency: %d\n", result.MaxConcurrency)

	// Output:
	// ErrorStrategy: FailFast
	// Timeout: 10s
	// MaxConcurrency: 100
}

func ExampleOrchestration() {
	mock := &mockOrchestration{}

	// Fluent API usage
	configured := mock.
		Named("user-processing").
		With(Config{Timeout: 30 * time.Second}).
		ErrorBoundary(CollectAll)

	fmt.Printf("Name: %s\n", configured.(*mockOrchestration).name)
	fmt.Printf("Timeout: %v\n", configured.(*mockOrchestration).config.Timeout)
	fmt.Printf("ErrorBoundary: %v\n", *configured.(*mockOrchestration).errorBoundary)

	// Output:
	// Name: user-processing
	// Timeout: 30s
	// ErrorBoundary: CollectAll
}

// ConfigBuilder Tests

func TestNewConfigBuilder(t *testing.T) {
	builder := NewConfigBuilder()

	if builder == nil {
		t.Fatal("NewConfigBuilder() returned nil")
	}

	if builder.config.ErrorStrategy != FailFast {
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
		ErrorStrategy:  CollectAll,
		Timeout:        60 * time.Second,
		Retries:        5,
		MaxConcurrency: 200,
		Context:        context.Background(),
	}

	builder := NewConfigBuilderFrom(base)

	if builder.config.ErrorStrategy != CollectAll {
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
		ErrorStrategy(CollectAll).
		Timeout(45 * time.Second).
		Retries(3).
		MaxConcurrency(150)

	// Should return the same builder instance
	if result != builder {
		t.Error("Fluent API should return the same builder instance")
	}

	// Verify values were set
	if builder.config.ErrorStrategy != CollectAll {
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
				return cb.ErrorStrategy(CollectAll).Timeout(30 * time.Second).MaxConcurrency(100)
			},
			expectError: false,
		},
		{
			name: "invalid error strategy",
			builderFunc: func(cb *ConfigBuilder) *ConfigBuilder {
				return cb.ErrorStrategy(ErrorStrategy(999))
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
	config := builder.ErrorStrategy(CollectAll).MustBuild()

	if config.ErrorStrategy != CollectAll {
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
	builder.ErrorStrategy(CollectAll).Timeout(60 * time.Second).Retries(5)

	// Add an error
	builder.MaxConcurrency(-1)

	// Reset should restore defaults
	builder.Reset()

	if builder.config.ErrorStrategy != FailFast {
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
	original.ErrorStrategy(CollectAll).Timeout(60 * time.Second)

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

func TestConfigBuilderHasErrors(t *testing.T) {
	builder := NewConfigBuilder()

	if builder.HasErrors() {
		t.Error("New builder should not have errors")
	}

	// Add an error
	builder.Timeout(-1 * time.Second)

	if !builder.HasErrors() {
		t.Error("Builder should have errors after invalid operation")
	}
}

func TestConfigBuilderErrors(t *testing.T) {
	builder := NewConfigBuilder()

	// No errors initially
	if errors := builder.Errors(); errors != nil {
		t.Error("New builder should return nil for Errors()")
	}

	// Add multiple errors
	builder.Timeout(-1 * time.Second)
	builder.MaxConcurrency(-1)

	errors := builder.Errors()
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	// Should return a copy
	errors[0] = fmt.Errorf("modified")
	newErrors := builder.Errors()
	if newErrors[0].Error() == "modified" {
		t.Error("Errors() should return a copy")
	}
}

func TestConfigBuilderWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	builder := NewConfigBuilder()
	config := builder.WithContext(ctx).MustBuild()

	if config.Context != ctx {
		t.Error("WithContext should set the context")
	}
}

func TestConfigBuilderComplexScenarios(t *testing.T) {
	// Test building multiple configs from same builder
	builder := NewConfigBuilder()

	// First config
	config1 := builder.Clone().ErrorStrategy(FailFast).Timeout(30 * time.Second).MustBuild()

	// Second config
	config2 := builder.Clone().ErrorStrategy(CollectAll).Timeout(60 * time.Second).MustBuild()

	if config1.ErrorStrategy == config2.ErrorStrategy {
		t.Error("Different configs should have different ErrorStrategy")
	}

	if config1.Timeout == config2.Timeout {
		t.Error("Different configs should have different Timeout")
	}
}

func TestConfigBuilderInheritanceIntegration(t *testing.T) {
	// Test that ConfigBuilder works well with Config.Inherit
	baseConfig := NewConfigBuilder().
		ErrorStrategy(FailFast).
		Timeout(30 * time.Second).
		MaxConcurrency(100).
		MustBuild()

	childConfig := NewConfigBuilderFrom(baseConfig).
		Timeout(10 * time.Second). // Override timeout
		MustBuild()

	// Test inheritance
	finalConfig := childConfig.Inherit(baseConfig)

	if finalConfig.ErrorStrategy != FailFast {
		t.Error("Should inherit ErrorStrategy")
	}

	if finalConfig.Timeout != 10*time.Second {
		t.Error("Should use child's timeout")
	}

	if finalConfig.MaxConcurrency != 100 {
		t.Error("Should inherit MaxConcurrency")
	}
}

// Benchmark tests for ConfigBuilder
func BenchmarkConfigBuilderBuild(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewConfigBuilder().
			ErrorStrategy(CollectAll).
			Timeout(30 * time.Second).
			Retries(3).
			MaxConcurrency(100).
			MustBuild()
	}
}

func BenchmarkConfigBuilderClone(b *testing.B) {
	builder := NewConfigBuilder().
		ErrorStrategy(CollectAll).
		Timeout(30 * time.Second).
		Retries(3).
		MaxConcurrency(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.Clone()
	}
}

func BenchmarkConfigBuilderReset(b *testing.B) {
	builder := NewConfigBuilder()

	for i := 0; i < b.N; i++ {
		builder.ErrorStrategy(CollectAll).
			Timeout(30 * time.Second).
			Retries(3).
			MaxConcurrency(100).
			Reset()
	}
}

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

// Helper function for string contains check
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

// Example tests for documentation
func ExampleNewConfigBuilder() {
	config := NewConfigBuilder().
		ErrorStrategy(CollectAll).
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
		ErrorStrategy(CollectAll).
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

func ExampleConfigBuilder_Clone() {
	baseBuilder := NewConfigBuilder().ErrorStrategy(CollectAll)

	fastConfig := baseBuilder.Clone().Timeout(5 * time.Second).MustBuild()
	slowConfig := baseBuilder.Clone().Timeout(60 * time.Second).MustBuild()

	fmt.Printf("Fast timeout: %v\n", fastConfig.Timeout)
	fmt.Printf("Slow timeout: %v\n", slowConfig.Timeout)
	fmt.Printf("Both have same ErrorStrategy: %v\n",
		fastConfig.ErrorStrategy == slowConfig.ErrorStrategy)

	// Output:
	// Fast timeout: 5s
	// Slow timeout: 1m0s
	// Both have same ErrorStrategy: true
}
