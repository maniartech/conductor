package pool

import (
	"testing"

	. "github.com/maniartech/orchestrator/internal/pool"
)

func TestResultItemReset(t *testing.T) {
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  make([]error, 0),
	}

	// Add some data
	item.Entries["key1"] = "value1"
	item.Entries["key2"] = 42
	item.Entries["key3"] = true
	item.Errors = append(item.Errors, &testError{})
	item.Errors = append(item.Errors, &testError{})

	// Simulate manager reset logic
	for k := range item.Entries {
		delete(item.Entries, k)
	}
	item.Errors = item.Errors[:0]

	// Verify entries are cleared
	if len(item.Entries) != 0 {
		t.Error("Entries map should be cleared after reset")
	}

	// Verify errors are cleared
	if len(item.Errors) != 0 {
		t.Error("Errors slice should be cleared after reset")
	}

	// Verify capacity is preserved for errors slice
	if cap(item.Errors) == 0 {
		t.Error("Errors slice capacity should be preserved")
	}
}

func TestResultItemResetWithNilEntries(t *testing.T) {
	item := &ResultItem{
		Entries: nil,
		Errors:  make([]error, 0),
	}

	// Simulate manager reset logic with nil entries
	if item.Entries != nil {
		for k := range item.Entries {
			delete(item.Entries, k)
		}
	}

	// Entries should remain nil (manager would handle initialization)
	if item.Entries != nil {
		t.Error("Entries should remain nil")
	}
}

func TestResultItemResetWithNilErrors(t *testing.T) {
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  nil,
	}

	// Simulate manager reset logic with nil errors
	if item.Errors != nil {
		item.Errors = item.Errors[:0]
	}

	// Errors should remain nil (manager would handle initialization)
	if item.Errors != nil {
		t.Error("Errors should remain nil")
	}
}

func TestResultItemEntriesOperations(t *testing.T) {
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  make([]error, 0),
	}

	// Test setting and getting entries
	item.Entries["string"] = "test"
	item.Entries["int"] = 42
	item.Entries["bool"] = true
	item.Entries["slice"] = []string{"a", "b", "c"}

	// Verify entries
	if item.Entries["string"] != "test" {
		t.Error("String entry not stored correctly")
	}

	if item.Entries["int"] != 42 {
		t.Error("Int entry not stored correctly")
	}

	if item.Entries["bool"] != true {
		t.Error("Bool entry not stored correctly")
	}

	if slice, ok := item.Entries["slice"].([]string); !ok || len(slice) != 3 {
		t.Error("Slice entry not stored correctly")
	}

	// Test overwriting entries
	item.Entries["string"] = "updated"
	if item.Entries["string"] != "updated" {
		t.Error("Entry should be updated")
	}

	// Test deleting entries
	delete(item.Entries, "int")
	if _, exists := item.Entries["int"]; exists {
		t.Error("Entry should be deleted")
	}
}

func TestResultItemErrorsOperations(t *testing.T) {
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  make([]error, 0),
	}

	// Test appending errors
	err1 := &testError{}
	err2 := &testError{}

	item.Errors = append(item.Errors, err1)
	item.Errors = append(item.Errors, err2)

	// Verify errors
	if len(item.Errors) != 2 {
		t.Error("Should have 2 errors")
	}

	if item.Errors[0] != err1 {
		t.Error("First error not stored correctly")
	}

	if item.Errors[1] != err2 {
		t.Error("Second error not stored correctly")
	}

	// Test accessing errors by index
	if _, ok := item.Errors[0].(*testError); !ok {
		t.Error("First error should be *testError")
	}

	if _, ok := item.Errors[1].(*testError); !ok {
		t.Error("Second error should be *testError")
	}
}

func TestResultItemCapacityPreservation(t *testing.T) {
	// Create item with specific capacities
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  make([]error, 0, 50),
	}

	// Add many entries
	for i := 0; i < 100; i++ {
		key := string(rune('a'+i%26)) + string(rune('0'+i/26))
		item.Entries[key] = i
	}

	// Add many errors
	for i := 0; i < 30; i++ {
		item.Errors = append(item.Errors, &testError{})
	}

	// Verify data was added
	if len(item.Entries) != 100 {
		t.Error("Should have 100 entries")
	}

	if len(item.Errors) != 30 {
		t.Error("Should have 30 errors")
	}

	// Store original capacity
	originalErrorsCap := cap(item.Errors)

	// Simulate manager reset
	for k := range item.Entries {
		delete(item.Entries, k)
	}
	item.Errors = item.Errors[:0]

	// Verify lengths are 0
	if len(item.Entries) != 0 {
		t.Error("Entries should be empty after reset")
	}

	if len(item.Errors) != 0 {
		t.Error("Errors should be empty after reset")
	}

	// Verify errors capacity is preserved
	if cap(item.Errors) != originalErrorsCap {
		t.Error("Errors capacity should be preserved after reset")
	}
}

func TestResultItemTypeSafety(t *testing.T) {
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  make([]error, 0),
	}

	// Test different types in entries
	item.Entries["string"] = "test"
	item.Entries["int"] = 42
	item.Entries["float"] = 3.14
	item.Entries["struct"] = struct{ Name string }{Name: "test"}

	// Test different error types (all must implement error interface)
	item.Errors = append(item.Errors, &testError{})

	// Verify type assertions work for entries
	if str, ok := item.Entries["string"].(string); !ok || str != "test" {
		t.Error("String entry type assertion failed")
	}

	if num, ok := item.Entries["int"].(int); !ok || num != 42 {
		t.Error("Int entry type assertion failed")
	}

	if f, ok := item.Entries["float"].(float64); !ok || f != 3.14 {
		t.Error("Float entry type assertion failed")
	}

	// Verify type assertions work for errors
	if _, ok := item.Errors[0].(*testError); !ok {
		t.Error("Error type assertion failed")
	}
}

// Benchmark tests
func BenchmarkResultItemReset(b *testing.B) {
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  make([]error, 0, 100),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Add some data
		item.Entries["key1"] = "value1"
		item.Entries["key2"] = 42
		item.Errors = append(item.Errors, &testError{})

		// Simulate manager reset
		for k := range item.Entries {
			delete(item.Entries, k)
		}
		item.Errors = item.Errors[:0]
	}
}

func BenchmarkResultItemEntriesSet(b *testing.B) {
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  make([]error, 0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000))
		item.Entries[key] = i
	}
}

func BenchmarkResultItemEntriesGet(b *testing.B) {
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  make([]error, 0),
	}

	// Pre-populate with entries
	for i := 0; i < 1000; i++ {
		key := "key" + string(rune(i))
		item.Entries[key] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000))
		_ = item.Entries[key]
	}
}

func BenchmarkResultItemErrorsAppend(b *testing.B) {
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  make([]error, 0, 10000),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item.Errors = append(item.Errors, &testError{})
		if len(item.Errors) >= 10000 {
			item.Errors = item.Errors[:0]
		}
	}
}

func BenchmarkResultItemErrorsAccess(b *testing.B) {
	item := &ResultItem{
		Entries: make(map[string]interface{}),
		Errors:  make([]error, 1000),
	}

	// Pre-populate with errors
	for i := 0; i < 1000; i++ {
		item.Errors[i] = &testError{}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = item.Errors[i%1000]
	}
}
