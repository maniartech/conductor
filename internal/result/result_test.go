package result

import (
	systemErrors "errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/errors"
)

// Test types for interface testing
type TestStringer interface {
	String() string
}

type TestMyString string

func (ms TestMyString) String() string {
	return string(ms)
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
	opErr := errors.OperationError{
		Error:     systemErrors.New("test error"),
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
					opErr := errors.OperationError{
						Error: systemErrors.New("test error"),
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

// TestResultMerge tests the Merge functionality
func TestResultMerge(t *testing.T) {
	t.Run("merge_basic", func(t *testing.T) {
		result1 := NewResult()
		result1.Set("key1", "value1")
		result1.Set("shared", "original")

		result2 := NewResult()
		result2.Set("key2", "value2")
		result2.Set("shared", "overwritten")

		result1.Merge(result2)

		// Check that all keys are present
		if result1.Get("key1") != "value1" {
			t.Errorf("Expected 'value1', got %v", result1.Get("key1"))
		}

		if result1.Get("key2") != "value2" {
			t.Errorf("Expected 'value2', got %v", result1.Get("key2"))
		}

		// Shared key should be overwritten
		if result1.Get("shared") != "overwritten" {
			t.Errorf("Expected 'overwritten', got %v", result1.Get("shared"))
		}
	})

	t.Run("merge_with_errors", func(t *testing.T) {
		result1 := NewResult()
		result1.AddError(errors.OperationError{
			Error: systemErrors.New("error1"),
			Index: 1,
		})

		result2 := NewResult()
		result2.AddError(errors.OperationError{
			Error: systemErrors.New("error2"),
			Index: 2,
		})

		result1.Merge(result2)

		resultErrors := result1.Errors()
		if len(resultErrors) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(resultErrors))
		}

		if resultErrors[0].Error.Error() != "error1" {
			t.Errorf("Expected 'error1', got %v", resultErrors[0].Error.Error())
		}

		if resultErrors[1].Error.Error() != "error2" {
			t.Errorf("Expected 'error2', got %v", resultErrors[1].Error.Error())
		}
	})

	t.Run("merge_nil_result", func(t *testing.T) {
		result := NewResult()
		result.Set("key", "value")

		// Merging nil should not panic or change anything
		result.Merge(nil)

		if result.Get("key") != "value" {
			t.Errorf("Expected 'value', got %v", result.Get("key"))
		}
	})

	t.Run("merge_empty_result", func(t *testing.T) {
		result1 := NewResult()
		result1.Set("key", "value")

		result2 := NewResult()

		result1.Merge(result2)

		if result1.Get("key") != "value" {
			t.Errorf("Expected 'value', got %v", result1.Get("key"))
		}
	})

	t.Run("merge_self", func(t *testing.T) {
		result := NewResult()
		result.Set("key", "value")
		result.AddError(errors.OperationError{
			Error: systemErrors.New("error"),
			Index: 1,
		})

		// Merging with self should be a no-op to prevent deadlock and duplication
		result.Merge(result)

		if result.Get("key") != "value" {
			t.Errorf("Expected 'value', got %v", result.Get("key"))
		}

		// Self-merge should not duplicate errors (should remain 1)
		resultErrors := result.Errors()
		if len(resultErrors) != 1 {
			t.Errorf("Expected 1 error after self-merge (no-op), got %d", len(resultErrors))
		}
	})
}

// TestResultConcurrentMerge tests concurrent merge operations
func TestResultConcurrentMerge(t *testing.T) {
	result1 := NewResult()
	result2 := NewResult()
	result3 := NewResult()

	// Populate results
	for i := 0; i < 100; i++ {
		result2.Set(fmt.Sprintf("key2-%d", i), i)
		result3.Set(fmt.Sprintf("key3-%d", i), i)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Concurrent merges
	go func() {
		defer wg.Done()
		result1.Merge(result2)
	}()

	go func() {
		defer wg.Done()
		result1.Merge(result3)
	}()

	wg.Wait()

	// Verify all keys are present
	for i := 0; i < 100; i++ {
		key2 := fmt.Sprintf("key2-%d", i)
		key3 := fmt.Sprintf("key3-%d", i)

		if result1.Get(key2) != i {
			t.Errorf("Expected %d for %s, got %v", i, key2, result1.Get(key2))
		}

		if result1.Get(key3) != i {
			t.Errorf("Expected %d for %s, got %v", i, key3, result1.Get(key3))
		}
	}
}

// TestResultEdgeCases tests edge cases and boundary conditions
func TestResultEdgeCases(t *testing.T) {
	t.Run("empty_key", func(t *testing.T) {
		result := NewResult()
		result.Set("", "empty_key_value")

		if result.Get("") != "empty_key_value" {
			t.Errorf("Expected 'empty_key_value', got %v", result.Get(""))
		}
	})

	t.Run("nil_value", func(t *testing.T) {
		result := NewResult()
		result.Set("nil_key", nil)

		if result.Get("nil_key") != nil {
			t.Errorf("Expected nil, got %v", result.Get("nil_key"))
		}

		// GetTyped with nil value
		if value, ok := GetTyped[any](result, "nil_key"); !ok {
			t.Error("GetTyped should succeed for nil value")
		} else if value != nil {
			t.Errorf("Expected nil, got %v", value)
		}
	})

	t.Run("overwrite_value", func(t *testing.T) {
		result := NewResult()
		result.Set("key", "original")
		result.Set("key", "overwritten")

		if result.Get("key") != "overwritten" {
			t.Errorf("Expected 'overwritten', got %v", result.Get("key"))
		}
	})

	t.Run("large_number_of_entries", func(t *testing.T) {
		result := NewResult()
		const numEntries = 10000

		// Set many entries
		for i := 0; i < numEntries; i++ {
			result.Set(fmt.Sprintf("key-%d", i), i)
		}

		// Verify all entries
		for i := 0; i < numEntries; i++ {
			key := fmt.Sprintf("key-%d", i)
			if result.Get(key) != i {
				t.Errorf("Expected %d for %s, got %v", i, key, result.Get(key))
			}
		}
	})

	t.Run("unicode_keys", func(t *testing.T) {
		result := NewResult()
		unicodeKey := "测试键"
		unicodeValue := "测试值"

		result.Set(unicodeKey, unicodeValue)

		if result.Get(unicodeKey) != unicodeValue {
			t.Errorf("Expected '%s', got %v", unicodeValue, result.Get(unicodeKey))
		}
	})
}

// TestResultComplexTypes tests with complex data types
func TestResultComplexTypes(t *testing.T) {
	result := NewResult()

	t.Run("nested_struct", func(t *testing.T) {
		type Address struct {
			Street string
			City   string
		}

		type Person struct {
			Name    string
			Age     int
			Address Address
		}

		person := Person{
			Name: "John Doe",
			Age:  30,
			Address: Address{
				Street: "123 Main St",
				City:   "Anytown",
			},
		}

		result.Set("person", person)

		if retrieved, ok := GetTyped[Person](result, "person"); !ok {
			t.Error("Should retrieve nested struct successfully")
		} else {
			if retrieved.Name != "John Doe" {
				t.Errorf("Expected 'John Doe', got %s", retrieved.Name)
			}
			if retrieved.Address.City != "Anytown" {
				t.Errorf("Expected 'Anytown', got %s", retrieved.Address.City)
			}
		}
	})

	t.Run("function_type", func(t *testing.T) {
		fn := func(x int) int { return x * 2 }
		result.Set("function", fn)

		if retrieved, ok := GetTyped[func(int) int](result, "function"); !ok {
			t.Error("Should retrieve function type successfully")
		} else {
			if retrieved(5) != 10 {
				t.Errorf("Expected function to return 10, got %d", retrieved(5))
			}
		}
	})

	t.Run("interface_with_methods", func(t *testing.T) {
		var s TestStringer = TestMyString("test")
		result.Set("stringer", s)

		if retrieved, ok := GetTyped[TestStringer](result, "stringer"); !ok {
			t.Error("Should retrieve interface type successfully")
		} else {
			if retrieved.String() != "test" {
				t.Errorf("Expected 'test', got %s", retrieved.String())
			}
		}
	})
}

// TestResultMemoryUsage tests memory efficiency
func TestResultMemoryUsage(t *testing.T) {
	t.Run("memory_reuse", func(t *testing.T) {
		result := NewResult()

		// Add and remove many entries to test memory reuse
		for i := 0; i < 1000; i++ {
			result.Set(fmt.Sprintf("temp-%d", i), i)
		}

		// Overwrite with fewer entries
		for i := 0; i < 10; i++ {
			result.Set(fmt.Sprintf("final-%d", i), i)
		}

		// The map should still work correctly
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("final-%d", i)
			if result.Get(key) != i {
				t.Errorf("Expected %d for %s, got %v", i, key, result.Get(key))
			}
		}
	})
}

// TestResultErrorHandling tests comprehensive error handling
func TestResultErrorHandling(t *testing.T) {
	t.Run("multiple_errors", func(t *testing.T) {
		result := NewResult()

		// Add multiple errors
		for i := 0; i < 5; i++ {
			result.AddError(errors.OperationError{
				Error:     systemErrors.New(fmt.Sprintf("error-%d", i)),
				Index:     i,
				Duration:  time.Duration(i) * time.Millisecond,
				Timestamp: time.Now().Add(time.Duration(i) * time.Second),
				OpID:      fmt.Sprintf("op-%d", i),
			})
		}

		if !result.HasErrors() {
			t.Error("Should have errors")
		}

		resultErrors := result.Errors()
		if len(resultErrors) != 5 {
			t.Errorf("Expected 5 errors, got %d", len(resultErrors))
		}

		// Verify error details
		for i, err := range resultErrors {
			expectedMsg := fmt.Sprintf("error-%d", i)
			if err.Error.Error() != expectedMsg {
				t.Errorf("Expected '%s', got '%s'", expectedMsg, err.Error.Error())
			}

			if err.Index != i {
				t.Errorf("Expected index %d, got %d", i, err.Index)
			}

			expectedOpID := fmt.Sprintf("op-%d", i)
			if err.OpID != expectedOpID {
				t.Errorf("Expected OpID '%s', got '%s'", expectedOpID, err.OpID)
			}
		}
	})

	t.Run("error_immutability", func(t *testing.T) {
		result := NewResult()
		originalError := errors.OperationError{
			Error: systemErrors.New("original"),
			Index: 1,
		}

		result.AddError(originalError)

		// Modify the original error
		originalError.Index = 999

		// The error in the result should not be affected
		resultErrors := result.Errors()
		if resultErrors[0].Index == 999 {
			t.Error("Result errors should not be affected by modifications to original")
		}
	})
}

// BenchmarkResultMerge benchmarks merge operations
func BenchmarkResultMerge(b *testing.B) {
	result1 := NewResult()
	result2 := NewResult()

	// Populate result2
	for i := 0; i < 100; i++ {
		result2.Set(fmt.Sprintf("key-%d", i), i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result1.Merge(result2)
	}
}

// BenchmarkResultAddError benchmarks error addition
func BenchmarkResultAddError(b *testing.B) {
	result := NewResult()
	err := errors.OperationError{
		Error: systemErrors.New("benchmark error"),
		Index: 1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.AddError(err)
	}
}
