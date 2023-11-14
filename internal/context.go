package internal

import (
	"context"
	"sync"
)

// ChoreoContext is a context implementation that allows values to be set and
// retrieved between activities. It ensures threadsafe get and set operations
// by using a mutex.
type ChoreoContext struct {
	context.Context
	values map[string]interface{}
	mu     sync.RWMutex
}

// Set sets the value in the context. Ensures thread safety by using a mutex.
func (c *ChoreoContext) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// Get returns the value from the context. Ensures thread safety by using a
// mutex.
func (c *ChoreoContext) Get(key string) (value interface{}, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok = c.values[key]
	return
}

// newContext creates a new context from the parent context. Panics if the
// parent context is nil. The panic is to ensure that the context is always
// created from a valid parent context.
func NewContext(parent context.Context) *ChoreoContext {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	return &ChoreoContext{
		parent,
		make(map[string]any),
		sync.RWMutex{},
	}

}
