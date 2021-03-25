package async

// Async
func Go(goFn ...interface{}) *Promise {
	return goExec(false, goFn...)
}

func GoQ(goFn ...interface{}) *Promise {
	return goExec(true, goFn...)
}

func goExec(q bool, args ...interface{}) *Promise {
	if len(args) == 0 {
		panic("arguments-missing")
	}

	if _, ok := args[0].(*Promise); ok {
		if q {
			return NewPromise(queuedProcessor, args...)
		}
		return NewPromise(parallalProcessor, args...)
	}
	return NewPromise(args[0].(func(*Promise, ...interface{})), args[1:]...)
}
