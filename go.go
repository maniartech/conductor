package choreo

// Go creates a new future which provides easy to await mechanism.
// It can be started either by using calling a `Start` or `Await` method.
//
//    func(fn ChoreographyHandler, args ...interface{}) *Promsie
//
// Example: Immediate start and await
//
//    // Starts a new process and awaits for it to finish.
//    v, err := async.Go(process, 1).Await()
//    if err != nil {
//      println("An error occurred while processing the future.")
//    }
//    print(v) // Print the resulted value
//
// Example: Delayed start
//
//    // Create a new future
//    p := async.Go(process, 1)
//    p.Then(func (v interface{}, e error) {
//      println("The process 1 finished.")
//    })
//
func Func(fn ChoreographyHandler, args ...interface{}) *Choreography {
	return create(fn, args...)
}

// Async creates a new future form list of futures and run them in parallel go routines.
// It returns the pointer to the newly created future.
//
//    //
//    func(futures ...*Choreography) *Choreography
//
// Example: (1)
//
//    async.GoC(
//      async.Go(process, 1),
//      async.Go(sendEmail, 2)
//    ).Await()
//
func Async(futures ...*Choreography) *Choreography {
	return createBatch(false, futures...)
}

// GoQ creates a new future form list of futures and run them in sequencial
// go routines! It returns the pointer to the newly created future.
//
// It accepts following function signatures.
//
//     1) func(futures ...*Choreography) *Choreography
//     2) func(hanlderFn ChoreographyHandler, args ...interface{}) *Promsie
//
// Example: (1)
//  async.Go(async.Go(process, 1), async.Go(sendEmail, 2))
//  async.Go(async.NewChoreography(process, 1), async.NewPromsie(process, 2))
//
// Example: (2)
//   async.Go(process, 1) // Just runs the go routine
//   async.Go(process, 2).Await() // Runs the go routine and await for it to finish.
//
func Sync(futures ...*Choreography) *Choreography {
	return createBatch(true, futures...)
}
