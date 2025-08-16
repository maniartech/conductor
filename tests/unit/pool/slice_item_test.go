package pool

import (
	"testing"

	. "github.com/maniartech/orchestrator/internal/pool"
)

func TestSliceItemReset(t *testing.T) {
	item := &SliceItem{
		Data: []interface{}{"a", "b", "c"},
	}

	// Simulate manager reset logic
	item.Data = item.Data[:0]

	// Verify reset
	if len(item.Data) != 0 {
		t.Error("Data slice should be reset to zero length")
	}

	// Capacity should be preserved
	if cap(item.Data) == 0 {
		t.Error("Data slice capacity should be preserved")
	}
}

func TestSliceItemResetWithNilSlice(t *testing.T) {
	item := &SliceItem{
		Data: nil,
	}

	// Simulate manager reset logic with nil slice
	if item.Data != nil {
		item.Data = item.Data[:0]
	}

	// Data should remain nil
	if item.Data != nil {
		t.Error("Data should remain nil after reset")
	}
}

func TestSliceItemCapacityPreservation(t *testing.T) {
	// Create item with specific capacity
	initialCap := 100
	item := &SliceItem{
		Data: make([]interface{}, 0, initialCap),
	}

	// Add some data
	for i := 0; i < 50; i++ {
		item.Data = append(item.Data, i)
	}

	// Verify data was added
	if len(item.Data) != 50 {
		t.Error("Data should have 50 elements")
	}

	// Simulate manager reset
	item.Data = item.Data[:0]

	// Verify length is 0 but capacity is preserved
	if len(item.Data) != 0 {
		t.Error("Length should be 0 after reset")
	}

	if cap(item.Data) != initialCap {
		t.Errorf("Capacity should be preserved: expected %d, got %d", initialCap, cap(item.Data))
	}
}

func TestSliceItemGrowth(t *testing.T) {
	item := &SliceItem{
		Data: make([]interface{}, 0, 10),
	}

	// Add more data than initial capacity
	for i := 0; i < 20; i++ {
		item.Data = append(item.Data, i)
	}

	// Verify data was added
	if len(item.Data) != 20 {
		t.Error("Data should have 20 elements")
	}

	// Capacity should have grown
	if cap(item.Data) < 20 {
		t.Error("Capacity should have grown to accommodate data")
	}

	// Simulate manager reset should preserve the grown capacity
	grownCap := cap(item.Data)
	item.Data = item.Data[:0]

	if cap(item.Data) != grownCap {
		t.Error("Reset should preserve grown capacity")
	}
}

func TestSliceItemTypeSafety(t *testing.T) {
	item := &SliceItem{
		Data: make([]interface{}, 0),
	}

	// Add different types
	item.Data = append(item.Data, "string")
	item.Data = append(item.Data, 42)
	item.Data = append(item.Data, true)
	item.Data = append(item.Data, []int{1, 2, 3})

	// Verify all types are stored
	if len(item.Data) != 4 {
		t.Error("Should store 4 different types")
	}

	// Verify type assertions work
	if str, ok := item.Data[0].(string); !ok || str != "string" {
		t.Error("String type assertion failed")
	}

	if num, ok := item.Data[1].(int); !ok || num != 42 {
		t.Error("Int type assertion failed")
	}

	if flag, ok := item.Data[2].(bool); !ok || !flag {
		t.Error("Bool type assertion failed")
	}

	if slice, ok := item.Data[3].([]int); !ok || len(slice) != 3 {
		t.Error("Slice type assertion failed")
	}
}

// Benchmark tests
func BenchmarkSliceItemReset(b *testing.B) {
	item := &SliceItem{
		Data: make([]interface{}, 100, 100),
	}

	// Fill with data
	for i := 0; i < 100; i++ {
		item.Data[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate manager reset
		item.Data = item.Data[:0]
		// Refill for next iteration
		for j := 0; j < 100; j++ {
			item.Data = append(item.Data, j)
		}
	}
}

func BenchmarkSliceItemAppend(b *testing.B) {
	item := &SliceItem{
		Data: make([]interface{}, 0, 1000),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item.Data = append(item.Data, i)
		if len(item.Data) >= 1000 {
			item.Data = item.Data[:0]
		}
	}
}

func BenchmarkSliceItemAccess(b *testing.B) {
	item := &SliceItem{
		Data: make([]interface{}, 1000),
	}

	// Fill with data
	for i := 0; i < 1000; i++ {
		item.Data[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = item.Data[i%1000]
	}
}
