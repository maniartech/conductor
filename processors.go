package async

import "sync"

type promiseProcessor func(*Promise, ...interface{})

var parallalProcessor func(*Promise, ...interface{}) = func(p *Promise, args ...interface{}) {
	wg := sync.WaitGroup{}

	for i := 0; i < len(args); i++ {
		pr, ok := args[i].(*Promise)
		if !ok {
			continue
		}
		wg.Add(1)
		pr.OnReady(func() {
			wg.Done()
		})
		pr.Go()
	}
	wg.Wait()
	p.Done()
}

var queuedProcessor func(*Promise, ...interface{}) = func(p *Promise, args ...interface{}) {
	for i := 0; i < len(args); i++ {
		pr, ok := args[i].(*Promise)
		if !ok {
			continue
		}
		pr.Await()
	}
	p.Done()
}
