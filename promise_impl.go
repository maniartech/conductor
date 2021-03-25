package async

import (
	"sync"
)

type ReadyHandler func()

// Async
func NewPromise(fn promiseProcessor, args ...interface{}) *Promise {
	return &Promise{

		Fn:   fn,
		Args: args,

		wg:            sync.WaitGroup{},
		readyHandlers: make([]ReadyHandler, 0),
	}
}

type Promise struct {
	// Fn represent the underlaying promised function
	Fn func(*Promise, ...interface{})

	// Args represents the arguments that needs to be passed when the promise is invoked
	Args []interface{}

	// Not Started: 0
	// Started: 1
	// Finished: 2
	status int
	wg     sync.WaitGroup

	readyHandlers []ReadyHandler

	// Result
	Result struct {
		Value interface{}
		Err   error
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
	go p.Fn(p, p.Args...)
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

func (p *Promise) Await() *Promise {
	// If the promise has already finished
	// do not wait further.
	if p.Finished() {
		return p
	}

	// The promise has not yet started, start it!
	if p.NotStarted() {
		p.Go()
	}

	p.wg.Wait()
	return p
}

// OnReady registers a new ReadyHandler function. The
// handler function is invoked when the promise
// has finished procesing.
func (p *Promise) OnReady(fn ReadyHandler) {
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
