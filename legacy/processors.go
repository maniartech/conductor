package orchestrator

import (
	"fmt"
	"sync"
)

// startAsync starts the specified futures in parallel goroutines.
func startAsync(p *Orchestration, args ...interface{}) {
	wg := sync.WaitGroup{}

	for i := 0; i < len(args); i++ {
		pr, ok := args[i].(*Orchestration)
		if !ok {
			panic(fmt.Errorf("%s at '%v'", errInvalidOrchestration, i))
		} else if pr.Pending() {
			panic(fmt.Errorf("%s at '%v'", errInvalidState, i))
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

// startSync starts the specified futures in new goroutines in a queued manner.
// That is, it starts the future only after the previous future has finished.
func startSync(p *Orchestration, args ...interface{}) {
	for i := 0; i < len(args); i++ {
		pr, ok := args[i].(*Orchestration)
		if !ok {
			panic(fmt.Errorf("%s at '%v'", errInvalidOrchestration, i))
		} else if pr.Pending() {
			panic(fmt.Errorf("%s at '%v'", errInvalidState, i))
		}
		pr.Await()
	}
	p.Done()
}

// Create creates a future task runner that executes a single task.
func create(fn OrchestrationHandler, args ...interface{}) *Orchestration {
	return &Orchestration{
		fn:   fn,
		args: args,
		wg:   sync.WaitGroup{},
	}
}

// createBatch creates a future that executes one or more handlers.
func createBatch(q bool, futures ...*Orchestration) *Orchestration {
	if len(futures) == 0 {
		panic(errInvalidArguments)
	}

	interfaces := make([]interface{}, len(futures))
	for i := 0; i < len(futures); i++ {
		interfaces[i] = futures[i]
	}

	var p *Orchestration

	if q {
		p = create(startSync, interfaces...)
	} else {
		p = create(startAsync, interfaces...)
	}
	p.batch = true
	return p
}
