package core

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

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

// Benchmark tests for Result
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

// Example tests for documentation
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
