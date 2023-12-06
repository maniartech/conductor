package choreo

import (
	"context"
	"errors"
	"sync"
)

type Player interface {
	Play(context.Context) (interface{}, error)
}

type HandlerFunc func(ctx context.Context) (interface{}, error)

func (f HandlerFunc) Start(ctx context.Context) (interface{}, error) {
	return f(ctx)
}

type Step interface {
	Name() string

	Start(ctx context.Context, args ...any) error

	OnStart(ActivityHandler)

	OnFinish(ActivityHandler)

	OnError(ErrorHandler)

	Await()
}

type Choreography struct {
	// choreographer *choreographer

	// Fn represent the underlaying futured function
	fn func(*Choreography, ...interface{})

	// Args represents the arguments that needs to be passed when the future is invoked
	args []interface{}

	// Not Started: 0
	// Pending: 1
	// Finished: 2
	status uint8

	wg sync.WaitGroup

	then ThenHandler

	// Result
	result interface{}

	// Error
	err error

	batch bool
}

// Start executes the future in the new go routine
func (c *Choreography) Start() {

	// Proceed only when the choreography is not started yet.
	if c.status != notStarted {
		return
	}

	// Add a wait group counter.
	c.wg.Add(1)

	c.status = pending

	// Execute the associated function in a new go routine
	go c.fn(c, c.args...)
}

// Done is designed to be executed by the
// invoker when the futured task is finished.
func (c *Choreography) Done(v ...interface{}) {
	for i := 0; i < len(v); i++ {
		if val, ok := v[i].(error); ok {
			c.err = val
		} else {
			c.result = v[i]
		}
	}
	c.wg.Done()
	c.status = finished

	// Invoke then function!
	if c.then != nil {
		c.then(c.result, c.err)
	}
}

// Await waits for future to finish and returns a resulting value.
func (c *Choreography) Await() (interface{}, error) {
	// If the future has already finished
	// do not wait further.
	if c.Finished() {
		return c.result, c.err
	}

	// The future has not yet started, start it!
	if c.NotStarted() {
		c.Start()
	}

	c.wg.Wait()
	return c.result, c.err
}

// Then is invoked when the associated future
// has finished procesing.
func (c *Choreography) Then(fn ThenHandler) {
	c.then = fn
}

// NotStarted returns `true` if the future exection has
// not yet started. Otherwise it returns `false`.
func (c *Choreography) NotStarted() bool {
	return c.status == notStarted
}

// Pending returns `true` if the future exection has
// not yet started. Otherwise it returns `false`.
func (c *Choreography) Pending() bool {
	return c.status == pending
}

// Finished returns `true` if the future has finished the
// function execution. It returns `false` otherwise.
func (c *Choreography) Finished() bool {
	return c.status == finished
}

// Result returns the value which is received after the successful
// execution of the associated function.
func (c *Choreography) Result() interface{} {
	return c.result
}

// Err returns the error that is reported when future has failed.
// When the future is successful or not yet finished, Err() will return `nil`.
func (c *Choreography) Err() error {
	return c.err
}

// Choreographys returns the associated child futures when created with GoP or GoQ functions!
// It returns nil otherwise.
func (c *Choreography) Choreographys() ([]*Choreography, error) {
	if !c.batch {
		return nil, errors.New(errNotABatch)
	}

	l := len(c.args)
	futures := make([]*Choreography, l)

	for i := 0; i < l; i++ {
		if future, ok := c.args[0].(*Choreography); ok {
			futures[i] = future
		} else {
			return nil, errors.New(errInvalidOperation)
		}
	}

	return futures, nil
}
