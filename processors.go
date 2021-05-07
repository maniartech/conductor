package async

import (
	"fmt"
	"sync"
)

// goConcurrent starts the specified promises in parallel go routines.
func goConcurrent(p *Promise, args ...interface{}) {
	wg := sync.WaitGroup{}

	for i := 0; i < len(args); i++ {
		pr, ok := args[i].(*Promise)
		if !ok {
			panic(fmt.Errorf("invalid promise at index '%v'", i))
		} else if pr.Pending() {
			panic(fmt.Errorf("the promise at index '%v' has already started", i))
		}
		wg.Add(1)
		pr.Then(func(interface{}, error) {
			wg.Done()
		})
		pr.Start()
	}
	wg.Wait()
	p.Done()
}

// goQueue starts the specified promises in new go routines with queued mannger.
// That is it starts the promise only after the preview promise has finished.
func goQueue(p *Promise, args ...interface{}) {
	for i := 0; i < len(args); i++ {
		pr, ok := args[i].(*Promise)
		if !ok {
			panic(fmt.Errorf("invalid promise at index '%v'", i))
		} else if pr.Pending() {
			panic(fmt.Errorf("the promise at index '%v' has already started", i))
		}
		pr.Await()
	}
	p.Done()
}

// Create a prmise that executes single handler
func create(fn PromiseHandler, args ...interface{}) *Promise {
	return &Promise{
		fn:   fn,
		args: args,
		wg:   sync.WaitGroup{},
	}
}

// Creates a promise that executes one or more handlers
func createBatch(q bool, promises ...*Promise) *Promise {
	if len(promises) == 0 {
		panic("arguments-missing")
	}

	interfaces := make([]interface{}, len(promises))
	for i := 0; i < len(promises); i++ {
		interfaces[i] = promises[i]
	}

	if q {
		return create(goQueue, interfaces...)
	}
	return create(goConcurrent, interfaces...)
}
