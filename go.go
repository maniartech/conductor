package async

// GoQ creates a new promise form list of promises and run them in parallel!
// It returns the pointer to the newly created promise.
//
// It accepts following function signatures.
//
// 1) func(promises ...*Promise) *Promise
//
// 2) func(hanlderFn PromiseHandler, args ...interface{}) *Promsie
//
// Example: (1)
// 	async.Go(async.Go(process, 1), async.Go(sendEmail, 2))
//  async.Go(async.NewPromise(process, 1), async.NewPromsie(process, 2))
//
// Example: (2)
//   async.Go(process, 1) // Just runs the go routine
//   async.Go(process, 2).Await() // Runs the go routine and await for it to finish.
//
func Go(goFn ...interface{}) *Promise {
	return goExec(false, goFn...)
}

// GoQ creates a new promise form list of promises and run them in sequencial
// order in queue! It returns the pointer to the newly created promise.
//
// It accepts following function signatures.
//
// 1) func(promises ...*Promise) *Promise
//
// 2) func(hanlderFn PromiseHandler, args ...interface{}) *Promsie
//
// Example: (1)
// 	async.Go(async.Go(process, 1), async.Go(sendEmail, 2))
//  async.Go(async.NewPromise(process, 1), async.NewPromsie(process, 2))
//
// Example: (2)
//   async.Go(process, 1) // Just runs the go routine
//   async.Go(process, 2).Await() // Runs the go routine and await for it to finish.
//
func GoQ(goFn ...interface{}) *Promise {
	return goExec(true, goFn...)
}
