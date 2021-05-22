package async

import (
	"errors"
	"sync"
)

// Promise status
const (
	notStarted uint8 = iota
	pending
	finished
)

// PromiseHandler provides a signature validation for
// promise function.
//
// Example:
//
//  func Process(p *Promise, ...v interface()) {
//    processId := v.(int)
//    result, err := SendRequest(processId)
//    // When finished processing, call Done by passing
//    // result and error details
//    p.Done(result, err)
//  }
type PromiseHandler func(*Promise, ...interface{})

// ThenHandler is a callback which will be
// invoked when the associated promise has finished.
//
// Example
//
//   p := async.Go(process, 1)
//   p.Then(func(v interface{}, e error) {
//     print("The promise has just finished!")
//   })
//
type ThenHandler func(interface{}, error)

type Promise struct {
	// Fn represent the underlaying promised function
	fn func(*Promise, ...interface{})

	// Args represents the arguments that needs to be passed when the promise is invoked
	args []interface{}

	// Not Started: 0
	// Pending: 1
	// Finished: 2
	status byte
	wg     sync.WaitGroup

	then ThenHandler
	// Result
	result interface{}

	// Error
	err error

	batch bool
}

// Start executes the promise in the new go routine
func (p *Promise) Start() {

	// Proceed only when the promise has not yet started.
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
// invoker when the promised task is finished.
func (p *Promise) Done(v ...interface{}) {
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

// Await waits for promise to finish and returns a resulting value.
func (p *Promise) Await() (interface{}, error) {
	// If the promise has already finished
	// do not wait further.
	if p.Finished() {
		return p.result, p.err
	}

	// The promise has not yet started, start it!
	if p.NotStarted() {
		p.Start()
	}

	p.wg.Wait()
	return p.result, p.err
}

// Then is invoked when the associated promise
// has finished procesing.
func (p *Promise) Then(fn ThenHandler) {
	p.then = fn
}

// NotStarted returns `true` if the promise exection has
// not yet started. Otherwise it returns `false`.
func (p *Promise) NotStarted() bool {
	return p.status == notStarted
}

// Pending returns `true` if the promise exection has
// not yet started. Otherwise it returns `false`.
func (p *Promise) Pending() bool {
	return p.status == pending
}

// Finished returns `true` if the promise has finished the
// function execution. It returns `false` otherwise.
func (p *Promise) Finished() bool {
	return p.status == finished
}

// Result returns the value which is received after the successful
// execution of the associated function.
func (p *Promise) Result() interface{} {
	return p.result
}

// Err returns the error that is reported when promise has failed.
// When the promise is successful or not yet finished, Err() will return `nil`.
func (p *Promise) Err() error {
	return p.err
}

// Promises returns the associated child promises when created with GoP or GoQ functions!
// It returns nil otherwise.
func (p *Promise) Promises() ([]*Promise, error) {
	if !p.batch {
		return nil, errors.New(errNotABatch)
	}

	l := len(p.args)
	promises := make([]*Promise, l)

	for i := 0; i < l; i++ {
		if promise, ok := p.args[0].(*Promise); ok {
			promises[i] = promise
		} else {
			return nil, errors.New(errInvalidOperation)
		}
	}
	return promises, nil
}
