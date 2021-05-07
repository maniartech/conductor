package async

// Go creates a new promise which provides easy to await mechanism.
// It can be started either by using calling a `Start` or `Await` method.
//
//    func(fn PromiseHandler, args ...interface{}) *Promsie
//
// Example: Immediate start and await
//
//    // Starts a new process and awaits for it to finish.
//    v, err := async.Go(process, 1).Await()
//    if err != nil {
//      println("An error occurred while processing the promise.")
//    }
//    print(v) // Print the resulted value
//
// Example: Delayed start
//
//    // Create a new promise
//    p := async.Go(process, 1)
//    p.Then(func (v interface{}, e error) {
//      println("The process 1 finished.")
//    })
//
func Go(fn PromiseHandler, args ...interface{}) *Promise {
	return create(fn, args...)
}

// GoC creates a new promise form list of promises and run them in parallel go routines.
// It returns the pointer to the newly created promise.
//
//    //
//    func(promises ...*Promise) *Promise
//
// Example: (1)
//
//    async.GoC(
//      async.Go(process, 1),
//      async.Go(sendEmail, 2)
//    ).Await()
//
func GoC(promises ...*Promise) *Promise {
	return createBatch(false, promises...)
}

// GoQ creates a new promise form list of promises and run them in sequencial
// go routines! It returns the pointer to the newly created promise.
//
// It accepts following function signatures.
//
//     1) func(promises ...*Promise) *Promise
//     2) func(hanlderFn PromiseHandler, args ...interface{}) *Promsie
//
// Example: (1)
//   async.Go(async.Go(process, 1), async.Go(sendEmail, 2))
//  async.Go(async.NewPromise(process, 1), async.NewPromsie(process, 2))
//
// Example: (2)
//   async.Go(process, 1) // Just runs the go routine
//   async.Go(process, 2).Await() // Runs the go routine and await for it to finish.
//
func GoQ(promises ...*Promise) *Promise {
	return createBatch(true, promises...)
}
