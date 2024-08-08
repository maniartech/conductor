package orchestrator

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

type Activity interface {
	Name() string

	Start(ctx context.Context, args ...any) error

	OnStart(ActivityHandler)

	OnFinish(ActivityHandler)

	OnError(ErrorHandler)

	Await()
}

type Orchestration struct {
	// Fn represent the underlying future function
	fn func(*Orchestration, ...interface{})

	// Args represents the arguments that need to be passed when the future is invoked
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

// Start executes the future in a new goroutine
func (o *Orchestration) Start() {

	// Proceed only when the orchestration has not started yet.
	if o.status != uint8(NotStarted) {

		return
	}

	// Add a wait group counter.
	o.wg.Add(1)

	o.status = uint8(Pending)

	// Execute the associated function in a new goroutine
	go o.fn(o, o.args...)
}

// Done is designed to be executed by the
// invoker when the future task is finished.
func (o *Orchestration) Done(v ...interface{}) {
	for i := 0; i < len(v); i++ {
		if val, ok := v[i].(error); ok {
			o.err = val
		} else {
			o.result = v[i]
		}
	}
	o.wg.Done()
	o.status = uint8(Finished)

	// Invoke then function!
	if o.then != nil {
		o.then(o.result, o.err)
	}
}

// Await waits for the future to finish and returns a resulting value.
func (o *Orchestration) Await() (interface{}, error) {
	// If the future has already finished
	// do not wait further.
	if o.Finished() {
		return o.result, o.err
	}

	// If the future has not yet started, start it!
	if o.NotStarted() {
		o.Start()
	}

	o.wg.Wait()
	return o.result, o.err
}

// Then is invoked when the associated future
// has finished processing.
func (o *Orchestration) Then(fn ThenHandler) {
	o.then = fn
}

// NotStarted returns `true` if the future execution has
// not yet started. Otherwise, it returns `false`.
func (o *Orchestration) NotStarted() bool {
	return o.status == uint8(NotStarted)
}

// Pending returns `true` if the future execution is
// in progress. Otherwise, it returns `false`.
func (o *Orchestration) Pending() bool {
	return o.status == uint8(Pending)
}

// Finished returns `true` if the future has finished the
// function execution. It returns `false` otherwise.
func (o *Orchestration) Finished() bool {
	return o.status == uint8(Finished)
}

// Result returns the value which is received after the successful
// execution of the associated function.
func (o *Orchestration) Result() interface{} {
	return o.result
}

// Err returns the error that is reported when the future has failed.
// When the future is successful or not yet finished, Err() will return `nil`.
func (o *Orchestration) Err() error {
	return o.err
}

// Orchestrations returns the associated child futures when created with GoP or GoQ functions!
// It returns nil otherwise.
func (o *Orchestration) Orchestrations() ([]*Orchestration, error) {
	if !o.batch {
		return nil, errors.New(errNotABatch)
	}

	l := len(o.args)
	futures := make([]*Orchestration, l)

	for i := 0; i < l; i++ {
		if future, ok := o.args[0].(*Orchestration); ok {
			futures[i] = future
		} else {
			return nil, errors.New(errInvalidOperation)
		}
	}

	return futures, nil
}
