package choreo

import (
	"context"
	"errors"
)

type choreoContext struct {
	context.Context

	values map[string]any
}

func (c *choreoContext) Get(key string, defaultValue ...any) (any, error) {
	if v, ok := c.values[key]; ok {
		return v, nil
	}

	if len(defaultValue) > 1 {
		return defaultValue, nil
	}

	return nil, errors.New("key-not-found: " + key)
}

func (c *choreoContext) Set(key string, value any) {
	c.values[key] = value
}

func newContext(parent context.Context) *choreoContext {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	return &choreoContext{
		parent,
		make(map[string]any),
	}

}

func init() {

	context.WithValue(context.Background(), "name", "teast")

}
