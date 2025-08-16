package pool

import (
	"testing"

	. "github.com/maniartech/orchestrator/internal/pool"
)

func TestContextItemFields(t *testing.T) {
	item := &ContextItem{
		Values: make(map[string]interface{}),
		Done:   make(chan struct{}),
	}

	// Test initial state
	if len(item.Values) != 0 {
		t.Error("Values map should be empty initially")
	}

	if item.Done == nil {
		t.Error("Done channel should not be nil")
	}

	// Add some values
	item.Values["key1"] = "value1"
	item.Values["key2"] = 42
	item.Values["key3"] = true

	// Verify values are set
	if len(item.Values) != 3 {
		t.Error("Values map should have 3 entries")
	}

	if item.Values["key1"] != "value1" {
		t.Error("key1 should be 'value1'")
	}

	// Test manual reset (simulating what manager does)
	for k := range item.Values {
		delete(item.Values, k)
	}

	if len(item.Values) != 0 {
		t.Error("Values map should be cleared after manual reset")
	}
}

func TestContextItemWithNilValues(t *testing.T) {
	item := &ContextItem{
		Values: nil,
		Done:   make(chan struct{}),
	}

	// Test that we can handle nil Values
	if item.Values != nil {
		t.Error("Values should be nil initially")
	}

	// Initialize values (simulating what manager does)
	if item.Values == nil {
		item.Values = make(map[string]interface{})
	}

	if item.Values == nil {
		t.Error("Values should be initialized")
	}

	if len(item.Values) != 0 {
		t.Error("Values should be empty after initialization")
	}
}

func TestContextItemWithNilDone(t *testing.T) {
	item := &ContextItem{
		Values: make(map[string]interface{}),
		Done:   nil,
	}

	// Test that we can handle nil Done
	if item.Done != nil {
		t.Error("Done should be nil initially")
	}

	// Initialize done channel (simulating what manager does)
	if item.Done == nil {
		item.Done = make(chan struct{})
	}

	if item.Done == nil {
		t.Error("Done should be initialized")
	}
}

func TestContextItemWithClosedChannel(t *testing.T) {
	item := &ContextItem{
		Values: make(map[string]interface{}),
		Done:   make(chan struct{}),
	}

	// Close the channel
	close(item.Done)

	// Verify channel is closed
	select {
	case <-item.Done:
		// Expected - channel is closed
	default:
		t.Error("Channel should be closed")
	}

	// Simulate manager creating new channel
	select {
	case <-item.Done:
		item.Done = make(chan struct{})
	default:
		// Channel is already fresh
	}

	// New channel should be open
	select {
	case <-item.Done:
		t.Error("New channel should not be closed")
	default:
		// Expected - new channel should be open
	}
}

func TestContextItemValueOperations(t *testing.T) {
	item := &ContextItem{
		Values: make(map[string]interface{}),
		Done:   make(chan struct{}),
	}

	// Test setting and getting values
	item.Values["string"] = "test"
	item.Values["int"] = 42
	item.Values["bool"] = true
	item.Values["slice"] = []string{"a", "b", "c"}

	// Verify values
	if item.Values["string"] != "test" {
		t.Error("String value not stored correctly")
	}

	if item.Values["int"] != 42 {
		t.Error("Int value not stored correctly")
	}

	if item.Values["bool"] != true {
		t.Error("Bool value not stored correctly")
	}

	if slice, ok := item.Values["slice"].([]string); !ok || len(slice) != 3 {
		t.Error("Slice value not stored correctly")
	}

	// Test overwriting values
	item.Values["string"] = "updated"
	if item.Values["string"] != "updated" {
		t.Error("Value should be updated")
	}

	// Test deleting values
	delete(item.Values, "int")
	if _, exists := item.Values["int"]; exists {
		t.Error("Value should be deleted")
	}
}

func TestContextItemMapCapacityPreservation(t *testing.T) {
	item := &ContextItem{
		Values: make(map[string]interface{}),
		Done:   make(chan struct{}),
	}

	// Add many values to potentially grow the map
	for i := 0; i < 100; i++ {
		key := string(rune('a'+i%26)) + string(rune('0'+i/26))
		item.Values[key] = i
	}

	// Verify all values are stored
	if len(item.Values) != 100 {
		t.Error("Should have 100 values")
	}

	// Simulate manager reset
	for k := range item.Values {
		delete(item.Values, k)
	}

	// Map should be empty but capacity might be preserved (implementation detail)
	if len(item.Values) != 0 {
		t.Error("Map should be empty after reset")
	}

	// Should be able to add values again
	item.Values["test"] = "value"
	if item.Values["test"] != "value" {
		t.Error("Should be able to add values after reset")
	}
}

func TestContextItemChannelOperations(t *testing.T) {
	item := &ContextItem{
		Values: make(map[string]interface{}),
		Done:   make(chan struct{}),
	}

	// Test that channel is initially open
	select {
	case <-item.Done:
		t.Error("Channel should be open initially")
	default:
		// Expected
	}

	// Test closing channel
	close(item.Done)

	// Verify channel is closed
	select {
	case <-item.Done:
		// Expected
	default:
		t.Error("Channel should be closed")
	}

	// Multiple reads from closed channel should work
	select {
	case <-item.Done:
		// Expected
	default:
		t.Error("Should be able to read from closed channel multiple times")
	}
}

// Benchmark tests
func BenchmarkContextItemReset(b *testing.B) {
	item := &ContextItem{
		Values: make(map[string]interface{}),
		Done:   make(chan struct{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Add some values
		item.Values["key1"] = "value1"
		item.Values["key2"] = 42
		item.Values["key3"] = true

		// Simulate manager reset
		for k := range item.Values {
			delete(item.Values, k)
		}
	}
}

func BenchmarkContextItemValueSet(b *testing.B) {
	item := &ContextItem{
		Values: make(map[string]interface{}),
		Done:   make(chan struct{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000))
		item.Values[key] = i
	}
}

func BenchmarkContextItemValueGet(b *testing.B) {
	item := &ContextItem{
		Values: make(map[string]interface{}),
		Done:   make(chan struct{}),
	}

	// Pre-populate with values
	for i := 0; i < 1000; i++ {
		key := "key" + string(rune(i))
		item.Values[key] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000))
		_ = item.Values[key]
	}
}

func BenchmarkContextItemChannelSelect(b *testing.B) {
	item := &ContextItem{
		Values: make(map[string]interface{}),
		Done:   make(chan struct{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case <-item.Done:
			// Should not happen in this benchmark
		default:
			// Expected path
		}
	}
}
