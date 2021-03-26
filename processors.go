package async

import "sync"

var goParallel func(*Promise, ...interface{}) = func(p *Promise, args ...interface{}) {
	wg := sync.WaitGroup{}

	for i := 0; i < len(args); i++ {
		pr, ok := args[i].(*Promise)
		if !ok {
			continue
		}
		wg.Add(1)
		pr.OnFinish(func() {
			wg.Done()
		})
		pr.Go()
	}
	wg.Wait()
	p.Done()
}

var goQueue func(*Promise, ...interface{}) = func(p *Promise, args ...interface{}) {
	for i := 0; i < len(args); i++ {
		pr, ok := args[i].(*Promise)
		if !ok {
			continue
		}
		pr.Await()
	}
	p.Done()
}

func goExec(q bool, args ...interface{}) *Promise {
	if len(args) == 0 {
		panic("arguments-missing")
	}

	if _, ok := args[0].(*Promise); ok {
		if q {
			return NewPromise(goQueue, args...)
		}
		return NewPromise(goParallel, args...)
	}
	return NewPromise(args[0].(func(*Promise, ...interface{})), args[1:]...)
}
