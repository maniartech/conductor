package async

import (
	"sync"
)

// PromiseHandler provides a signature validation for
// promise function.
//
// Example:
//
// func Process(p *Promise, ...v interface()) {
//   processId := v.(int)
//   result, err := SendRequest(processId)
//   // When finished processing, call Done by passing
//   // result and error details
//   p.Done(result, err)
// }
type PromiseHandler func(*Promise, ...interface{})

// FinishHandler is used to register a callback which aims to be
// invoked when the associated promise is finished processing the
// promised function.
//
// Example
//
// promise := async.NewPromise(process, 1)
// promise.OnFinish(func() {
//     print("The promise has just finished!")
// })
//
type FinishHandler func()

type Promise struct {
	// Fn represent the underlaying promised function
	fn func(*Promise, ...interface{})

	// Args represents the arguments that needs to be passed when the promise is invoked
	args []interface{}

	// Not Started: 0
	// Started: 1
	// Finished: 2
	status int
	wg     sync.WaitGroup

	readyHandlers []FinishHandler

	// Result
	Result struct {
		Value interface{}
		Err   error
	}
}

// NewPromise creates a new Promise. It does not start it however.
// To start the promise use Promise.Go method.
//
// For Example:
//
// p := NewPromise(processAsync, 1)
// p.Go()
//
func NewPromise(fn PromiseHandler, args ...interface{}) *Promise {
	return &Promise{
		fn:            fn,
		args:          args,
		wg:            sync.WaitGroup{},
		readyHandlers: make([]FinishHandler, 0),
	}
}

// Go executes the promise in the
// new go routine
func (p *Promise) Go() *Promise {

	// Proceed further only when the promise has
	// not started.
	if !p.NotStarted() {
		return p
	}

	// Add waitGroup
	p.wg.Add(1)
	p.status = 1
	go p.fn(p, p.args...)
	return p
}

// Done is designed to be executed by the
// invoker when the promised task is finished.
func (p *Promise) Done(v ...interface{}) {
	for i := 0; i < len(v); i++ {
		if val, ok := v[i].(error); ok {
			p.Result.Err = val
		} else {
			p.Result.Value = val
		}
	}
	p.wg.Done()
	p.status = 2 // Finished

	// Notify ready handlers!
	if len(p.readyHandlers) != 0 {
		for i := 0; i < len(p.readyHandlers); i++ {
			p.readyHandlers[i]()
		}
	}
}

func (p *Promise) Await() {
	// If the promise has already finished
	// do not wait further.
	if p.Finished() {
		return
	}

	// The promise has not yet started, start it!
	if p.NotStarted() {
		p.Go()
	}

	p.wg.Wait()
}

// OnFinish registers a new FinishHandler function. The
// handler function is invoked when the promise
// has finished procesing.
func (p *Promise) OnFinish(fn FinishHandler) {
	p.readyHandlers = append(p.readyHandlers, fn)
}

// NotStarted returns `true` if the promise exection has
// not yet started.It returns `false`.
func (p *Promise) NotStarted() bool {
	return p.status == 0
}

// Started returns `true` if the promise exection has started.
// It returns `false` otherwise.
func (p *Promise) Started() bool {
	return p.status == 1
}

// Finished returns `true` if the promise has finished the
// function execution. It returns `false` otherwise.
func (p *Promise) Finished() bool {
	return p.status == 2
}
