// Package atomic provides atomic operation utilities for high-performance
// orchestration operations with zero-allocation hot paths.
//
// This package implements military-grade atomic operations following KISS
// principles while providing enterprise-grade performance optimizations.
package atomic

import (
	"sync"
	"sync/atomic"
)

// Counter provides an atomic counter with additional utility methods.
type Counter struct {
	value atomic.Int64
}

// NewCounter creates a new atomic counter initialized to zero.
func NewCounter() *Counter {
	return &Counter{}
}

// NewCounterWithValue creates a new atomic counter with the specified initial value.
func NewCounterWithValue(initial int64) *Counter {
	c := &Counter{}
	c.value.Store(initial)
	return c
}

// Add atomically adds delta to the counter and returns the new value.
func (c *Counter) Add(delta int64) int64 {
	return c.value.Add(delta)
}

// Inc atomically increments the counter by 1 and returns the new value.
func (c *Counter) Inc() int64 {
	return c.value.Add(1)
}

// Dec atomically decrements the counter by 1 and returns the new value.
func (c *Counter) Dec() int64 {
	return c.value.Add(-1)
}

// Load atomically loads and returns the current value.
func (c *Counter) Load() int64 {
	return c.value.Load()
}

// Store atomically stores the given value.
func (c *Counter) Store(val int64) {
	c.value.Store(val)
}

// CompareAndSwap atomically compares the current value with old and
// swaps it with new if they are equal. Returns true if the swap occurred.
func (c *Counter) CompareAndSwap(old, new int64) bool {
	return c.value.CompareAndSwap(old, new)
}

// Reset atomically resets the counter to zero and returns the previous value.
func (c *Counter) Reset() int64 {
	return c.value.Swap(0)
}

// Flag provides an atomic boolean flag with utility methods.
type Flag struct {
	value atomic.Bool
}

// NewFlag creates a new atomic flag initialized to false.
func NewFlag() *Flag {
	return &Flag{}
}

// NewFlagWithValue creates a new atomic flag with the specified initial value.
func NewFlagWithValue(initial bool) *Flag {
	f := &Flag{}
	f.value.Store(initial)
	return f
}

// Set atomically sets the flag to true.
func (f *Flag) Set() {
	f.value.Store(true)
}

// Clear atomically sets the flag to false.
func (f *Flag) Clear() {
	f.value.Store(false)
}

// IsSet atomically loads and returns true if the flag is set.
func (f *Flag) IsSet() bool {
	return f.value.Load()
}

// Toggle atomically toggles the flag and returns the new value.
func (f *Flag) Toggle() bool {
	for {
		old := f.value.Load()
		new := !old
		if f.value.CompareAndSwap(old, new) {
			return new
		}
	}
}

// CompareAndSwap atomically compares the current value with old and
// swaps it with new if they are equal. Returns true if the swap occurred.
func (f *Flag) CompareAndSwap(old, new bool) bool {
	return f.value.CompareAndSwap(old, new)
}

// SetOnce atomically sets the flag to true only if it's currently false.
// Returns true if the flag was set, false if it was already true.
func (f *Flag) SetOnce() bool {
	return f.value.CompareAndSwap(false, true)
}

// Pointer provides an atomic pointer with type safety.
type Pointer[T any] struct {
	value atomic.Pointer[T]
}

// NewPointer creates a new atomic pointer initialized to nil.
func NewPointer[T any]() *Pointer[T] {
	return &Pointer[T]{}
}

// NewPointerWithValue creates a new atomic pointer with the specified initial value.
func NewPointerWithValue[T any](initial *T) *Pointer[T] {
	p := &Pointer[T]{}
	p.value.Store(initial)
	return p
}

// Load atomically loads and returns the current pointer value.
func (p *Pointer[T]) Load() *T {
	return p.value.Load()
}

// Store atomically stores the given pointer value.
func (p *Pointer[T]) Store(val *T) {
	p.value.Store(val)
}

// CompareAndSwap atomically compares the current pointer with old and
// swaps it with new if they are equal. Returns true if the swap occurred.
func (p *Pointer[T]) CompareAndSwap(old, new *T) bool {
	return p.value.CompareAndSwap(old, new)
}

// Swap atomically stores new and returns the previous value.
func (p *Pointer[T]) Swap(new *T) *T {
	return p.value.Swap(new)
}

// Status provides atomic status management with predefined states.
type Status struct {
	value atomic.Uint32
}

// Common status values
const (
	StatusNotStarted uint32 = iota
	StatusRunning
	StatusCompleted
	StatusCancelled
	StatusFailed
)

// NewStatus creates a new atomic status initialized to NotStarted.
func NewStatus() *Status {
	return &Status{}
}

// NewStatusWithValue creates a new atomic status with the specified initial value.
func NewStatusWithValue(initial uint32) *Status {
	s := &Status{}
	s.value.Store(initial)
	return s
}

// Load atomically loads and returns the current status.
func (s *Status) Load() uint32 {
	return s.value.Load()
}

// Store atomically stores the given status.
func (s *Status) Store(val uint32) {
	s.value.Store(val)
}

// CompareAndSwap atomically compares the current status with old and
// swaps it with new if they are equal. Returns true if the swap occurred.
func (s *Status) CompareAndSwap(old, new uint32) bool {
	return s.value.CompareAndSwap(old, new)
}

// IsRunning atomically checks if the status is Running.
func (s *Status) IsRunning() bool {
	return s.value.Load() == uint32(StatusRunning)
}

// IsCompleted atomically checks if the status is in a terminal state.
func (s *Status) IsCompleted() bool {
	status := s.value.Load()
	return status == uint32(StatusCompleted) ||
		status == uint32(StatusCancelled) ||
		status == uint32(StatusFailed)
}

// MemoryBarrier provides memory barrier utilities for proper synchronization.
type MemoryBarrier struct{}

// LoadBarrier ensures that all loads before this point complete before
// any loads after this point begin.
func (MemoryBarrier) LoadBarrier() {
	// Use atomic load as a memory barrier
	var dummy int32
	atomic.LoadInt32(&dummy)
}

// StoreBarrier ensures that all stores before this point complete before
// any stores after this point begin.
func (MemoryBarrier) StoreBarrier() {
	// Use atomic store as a memory barrier
	var dummy int32
	atomic.StoreInt32(&dummy, 0)
}

// FullBarrier ensures that all memory operations before this point complete
// before any memory operations after this point begin.
func (MemoryBarrier) FullBarrier() {
	// Use atomic add as a full memory barrier
	var dummy int32
	atomic.AddInt32(&dummy, 0)
}

// Metrics provides atomic metrics collection with zero-allocation hot paths.
type Metrics struct {
	counters map[string]*Counter
	flags    map[string]*Flag
	mu       sync.RWMutex // Protects the maps
}

// NewMetrics creates a new atomic metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		counters: make(map[string]*Counter),
		flags:    make(map[string]*Flag),
	}
}

// Counter gets or creates an atomic counter with the given name.
func (m *Metrics) Counter(name string) *Counter {
	// Try read lock first for fast path
	m.mu.RLock()
	if counter, exists := m.counters[name]; exists {
		m.mu.RUnlock()
		return counter
	}
	m.mu.RUnlock()

	// Need to create new counter, use write lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if counter, exists := m.counters[name]; exists {
		return counter
	}

	counter := NewCounter()
	m.counters[name] = counter
	return counter
}

// Flag gets or creates an atomic flag with the given name.
func (m *Metrics) Flag(name string) *Flag {
	// Try read lock first for fast path
	m.mu.RLock()
	if flag, exists := m.flags[name]; exists {
		m.mu.RUnlock()
		return flag
	}
	m.mu.RUnlock()

	// Need to create new flag, use write lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if flag, exists := m.flags[name]; exists {
		return flag
	}

	flag := NewFlag()
	m.flags[name] = flag
	return flag
}

// GetCounterValue atomically loads the value of the named counter.
// Returns 0 if the counter doesn't exist.
func (m *Metrics) GetCounterValue(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if counter, exists := m.counters[name]; exists {
		return counter.Load()
	}
	return 0
}

// GetFlagValue atomically loads the value of the named flag.
// Returns false if the flag doesn't exist.
func (m *Metrics) GetFlagValue(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if flag, exists := m.flags[name]; exists {
		return flag.IsSet()
	}
	return false
}

// Reset atomically resets all counters to zero and all flags to false.
func (m *Metrics) Reset() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, counter := range m.counters {
		counter.Reset()
	}
	for _, flag := range m.flags {
		flag.Clear()
	}
}

// LockFreeMap provides a lock-free map implementation for simple use cases.
// Note: This is a simplified implementation for demonstration.
// In production, consider using a more sophisticated lock-free map.
type LockFreeMap[K comparable, V any] struct {
	data atomic.Pointer[map[K]V]
}

// NewLockFreeMap creates a new lock-free map.
func NewLockFreeMap[K comparable, V any]() *LockFreeMap[K, V] {
	m := &LockFreeMap[K, V]{}
	m.data.Store(&map[K]V{})
	return m
}

// Load atomically loads the value for the given key.
// Returns the value and true if the key exists, zero value and false otherwise.
func (m *LockFreeMap[K, V]) Load(key K) (V, bool) {
	data := m.data.Load()
	if data == nil {
		var zero V
		return zero, false
	}

	value, exists := (*data)[key]
	return value, exists
}

// Store atomically stores the key-value pair.
// Note: This creates a new map copy, so it's not suitable for high-frequency updates.
func (m *LockFreeMap[K, V]) Store(key K, value V) {
	for {
		oldData := m.data.Load()
		if oldData == nil {
			oldData = &map[K]V{}
		}

		// Create a new map with the updated value
		newData := make(map[K]V, len(*oldData)+1)
		for k, v := range *oldData {
			newData[k] = v
		}
		newData[key] = value

		// Try to swap the new map
		if m.data.CompareAndSwap(oldData, &newData) {
			break
		}
	}
}

// Delete atomically deletes the key from the map.
func (m *LockFreeMap[K, V]) Delete(key K) {
	for {
		oldData := m.data.Load()
		if oldData == nil {
			return
		}

		// Check if key exists
		if _, exists := (*oldData)[key]; !exists {
			return
		}

		// Create a new map without the key
		newData := make(map[K]V, len(*oldData)-1)
		for k, v := range *oldData {
			if k != key {
				newData[k] = v
			}
		}

		// Try to swap the new map
		if m.data.CompareAndSwap(oldData, &newData) {
			break
		}
	}
}

// Len atomically returns the number of key-value pairs in the map.
func (m *LockFreeMap[K, V]) Len() int {
	data := m.data.Load()
	if data == nil {
		return 0
	}
	return len(*data)
}

// Range atomically iterates over all key-value pairs in the map.
// The function f is called for each pair. If f returns false, iteration stops.
func (m *LockFreeMap[K, V]) Range(f func(key K, value V) bool) {
	data := m.data.Load()
	if data == nil {
		return
	}

	// Create a snapshot to iterate over
	for k, v := range *data {
		if !f(k, v) {
			break
		}
	}
}
