package async

import (
	"errors"
	"sync"
)

// Future status
const (
	notStarted uint8 = iota
	pending
	finished
)

// FutureHandler provides a signature validation for
// future function.
//
// Example:
//
//  func Process(p *Future, ...v interface()) {
//    processId := v.(int)
//    result, err := SendRequest(processId)
//    // When finished processing, call Done by passing
//    // result and error details
//    p.Done(result, err)
//  }
type FutureHandler func(*Future, ...interface{})

// ThenHandler is a callback which will be
// invoked when the associated future has finished.
//
// Example
//
//   p := async.Go(process, 1)
//   p.Then(func(v interface{}, e error) {
//     print("The future has just finished!")
//   })
//
type ThenHandler func(interface{}, error)

type Future struct {
	// Fn represent the underlaying futured function
	fn func(*Future, ...interface{})

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
func (p *Future) Start() {

	// Proceed only when the future has not yet started.
	if p.status != notStarted {
		return
	}

	// Add a wait group counter.
	p.wg.Add(1)

	p.status = pending

	// Execute the associated function in a new go routine
	go p.fn(p, p.args...)
}

// Done is designed to be executed by the
// invoker when the futured task is finished.
func (p *Future) Done(v ...interface{}) {
	for i := 0; i < len(v); i++ {
		if val, ok := v[i].(error); ok {
			p.err = val
		} else {
			p.result = v[i]
		}
	}
	p.wg.Done()
	p.status = finished

	// Invoke then function!
	if p.then != nil {
		p.then(p.result, p.err)
	}
}

// Await waits for future to finish and returns a resulting value.
func (p *Future) Await() (interface{}, error) {
	// If the future has already finished
	// do not wait further.
	if p.Finished() {
		return p.result, p.err
	}

	// The future has not yet started, start it!
	if p.NotStarted() {
		p.Start()
	}

	p.wg.Wait()
	return p.result, p.err
}

// Then is invoked when the associated future
// has finished procesing.
func (p *Future) Then(fn ThenHandler) {
	p.then = fn
}

// NotStarted returns `true` if the future exection has
// not yet started. Otherwise it returns `false`.
func (p *Future) NotStarted() bool {
	return p.status == notStarted
}

// Pending returns `true` if the future exection has
// not yet started. Otherwise it returns `false`.
func (p *Future) Pending() bool {
	return p.status == pending
}

// Finished returns `true` if the future has finished the
// function execution. It returns `false` otherwise.
func (p *Future) Finished() bool {
	return p.status == finished
}

// Result returns the value which is received after the successful
// execution of the associated function.
func (p *Future) Result() interface{} {
	return p.result
}

// Err returns the error that is reported when future has failed.
// When the future is successful or not yet finished, Err() will return `nil`.
func (p *Future) Err() error {
	return p.err
}

// Futures returns the associated child futures when created with GoP or GoQ functions!
// It returns nil otherwise.
func (p *Future) Futures() ([]*Future, error) {
	if !p.batch {
		return nil, errors.New(errNotABatch)
	}

	l := len(p.args)
	futures := make([]*Future, l)

	for i := 0; i < l; i++ {
		if future, ok := p.args[0].(*Future); ok {
			futures[i] = future
		} else {
			return nil, errors.New(errInvalidOperation)
		}
	}

	return futures, nil
}
